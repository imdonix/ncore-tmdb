package service

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"ncore-tmdb/internal/database"
)

// Movie resolutions in display order (best first).
var movieResolutionOrder = []string{"2160p", "1080p", "720p", "480p", "SD"}

// Order used when the caller does not pick a resolution (1080p is the sweet spot).
var movieDefaultResolutionOrder = []string{"1080p", "2160p", "720p", "480p", "SD"}

// MovieDetails is the compact TMDB movie info shown in the widget.
type MovieDetails struct {
	TMDBID        int    `json:"tmdbId"`
	Name          string `json:"name"`
	Year          string `json:"year"`
	ReleaseDate   string `json:"releaseDate"`
	PosterPath    string `json:"posterPath,omitempty"`
	SearchPattern string `json:"searchPattern"`
}

// MovieResolution is one resolution nCore has releases for.
type MovieResolution struct {
	Quality  string `json:"quality"`
	Releases int    `json:"releases"`
	Seeders  int    `json:"seeders"` // best release's seeders
}

// MovieOverview is everything the movie widget needs for its first render.
type MovieOverview struct {
	Movie       MovieDetails        `json:"movie"`
	Resolutions []MovieResolution   `json:"resolutions"`
	Download    *MovieDownloadState `json:"download,omitempty"`
	SearchError string              `json:"searchError,omitempty"`
}

// MovieDownloadState is the stored download enriched with qBittorrent progress.
type MovieDownloadState struct {
	*database.MovieDownload
	Progress   float64 `json:"progress"` // 0..100
	State      string  `json:"state,omitempty"`
	Size       int64   `json:"size,omitempty"`
	Downloaded int64   `json:"downloaded,omitempty"`
	Dlspeed    int64   `json:"dlspeed,omitempty"`
	Eta        int64   `json:"eta,omitempty"`
	Ratio      float64 `json:"ratio,omitempty"`
}

var (
	movieDownloadMu sync.Mutex
	movieYearRe     = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)

	movieReleaseCacheMu sync.Mutex
	movieReleaseCache   = map[string]movieReleaseCacheEntry{}
)

const movieReleaseCacheTTL = 10 * time.Minute

type movieReleaseCacheEntry struct {
	releases []Torrent
	at       time.Time
}

// GetMovieOverview loads TMDB details, the available nCore resolutions, and any
// existing qBittorrent download for the movie.
func GetMovieOverview(tmdbID int) (*MovieOverview, error) {
	movie, err := getMovieDetails(tmdbID)
	if err != nil {
		return nil, err
	}

	overview := &MovieOverview{
		Movie:       movie,
		Resolutions: []MovieResolution{},
	}

	releases, searchErr := cachedMovieReleases(movie.Name, movie.Year)
	if searchErr != nil {
		overview.SearchError = searchErr.Error()
		log.Printf("movie: ncore search %s (%d): %v", movie.Name, tmdbID, searchErr)
	} else {
		overview.Resolutions = buildMovieResolutions(releases)
	}

	if download, err := GetMovieDownloadState(tmdbID); err == nil {
		overview.Download = download
	}

	return overview, nil
}

// GetMovieDownloadState returns the movie's stored download (if any) with live
// qBittorrent progress. A nil result means nothing has been downloaded yet.
func GetMovieDownloadState(tmdbID int) (*MovieDownloadState, error) {
	rec, err := database.GetMovieDownload(tmdbID)
	if err != nil || rec == nil {
		return nil, err
	}
	return movieDownloadState(rec), nil
}

// DownloadMovie picks the best release for the requested resolution and sends it
// to qBittorrent. An empty quality selects the best available resolution.
func DownloadMovie(tmdbID int, quality string) (*MovieDownloadState, error) {
	movieDownloadMu.Lock()
	defer movieDownloadMu.Unlock()

	movie, err := getMovieDetails(tmdbID)
	if err != nil {
		return nil, err
	}

	releases, err := cachedMovieReleases(movie.Name, movie.Year)
	if err != nil {
		return nil, fmt.Errorf("ncore search: %w", err)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found on nCore for %s", movie.Name)
	}

	if quality == "" {
		quality = bestAvailableQuality(releases)
		if quality == "" {
			quality = "Auto"
		}
	}

	best, ok := bestMovieRelease(releases, quality)
	if !ok {
		return nil, fmt.Errorf("no %s release found on nCore for %s", quality, movie.Name)
	}

	// Never add a second copy if this movie is already in qBittorrent.
	if existing, _ := database.GetMovieDownload(tmdbID); existing != nil {
		if qt, _ := GetTorrentByNcoreID(existing.NCoreTorrentID); qt != nil {
			return movieDownloadState(existing), nil
		}
		// qBittorrent can take a moment to index a new torrent. Avoid a
		// duplicate add if the same request arrives twice in quick succession.
		if existing.Status == "downloading" {
			if created, err := time.Parse(time.RFC3339, existing.CreatedAt); err == nil &&
				time.Since(created) < 2*time.Minute {
				return movieDownloadState(existing), nil
			}
		}
	}

	data, err := DownloadTorrent(best.ID)
	if err != nil {
		return nil, fmt.Errorf("download torrent: %w", err)
	}

	filename := best.Title
	if filename == "" {
		filename = best.ID
	}

	// qBittorrent's default save path + a per-movie folder keeps Jellyfin happy.
	folder := MovieFolderName(movie.Name, movie.Year)
	if err := AddTorrent(data, filename+".torrent", AddTorrentOpts{
		NcoreID:  best.ID,
		SavePath: folder,
	}); err != nil {
		return nil, fmt.Errorf("qbit: %w", err)
	}

	qbitHash := ""
	if qt, err := GetTorrentByNcoreID(best.ID); err == nil && qt != nil {
		qbitHash = qt.Hash
	}

	rec := &database.MovieDownload{
		TMDBID:         tmdbID,
		NCoreTorrentID: best.ID,
		TorrentTitle:   best.Title,
		Quality:        quality,
		QbitHash:       qbitHash,
		Status:         "downloading",
	}
	if err := database.UpsertMovieDownload(rec); err != nil {
		return nil, err
	}
	_ = database.UpdateContentProgress(tmdbID, "movie", "DOWNLOADING")

	return movieDownloadState(rec), nil
}

func movieDownloadState(rec *database.MovieDownload) *MovieDownloadState {
	state := &MovieDownloadState{MovieDownload: rec}
	if rec.Status == "" {
		rec.Status = "downloading"
	}

	qt, err := GetTorrentByNcoreID(rec.NCoreTorrentID)
	if err != nil {
		// qBittorrent is unreachable; report the last stored status.
		return state
	}
	if qt == nil {
		// qBittorrent is reachable but the torrent is gone.
		if rec.Status != "completed" {
			rec.Status = "missing"
		}
		return state
	}

	rec.QbitHash = qt.Hash
	state.State = qt.State
	state.Size = qt.Size
	state.Downloaded = qt.Downloaded
	state.Dlspeed = qt.Dlspeed
	state.Eta = qt.Eta
	state.Ratio = qt.Ratio
	state.Progress = qt.Progress * 100

	if qt.Progress >= 0.999 {
		rec.Status = "completed"
		state.Progress = 100
	} else {
		rec.Status = "downloading"
	}
	return state
}

func getMovieDetails(tmdbID int) (MovieDetails, error) {
	details, err := GetMovieDetailsTMDB(tmdbID)
	if err != nil {
		return MovieDetails{}, fmt.Errorf("tmdb: %w", err)
	}

	name := strings.TrimSpace(fmt.Sprintf("%v", details["title"]))
	if name == "" || name == "<nil>" {
		return MovieDetails{}, fmt.Errorf("tmdb: movie %d has no title", tmdbID)
	}

	releaseDate, _ := details["release_date"].(string)
	year := ""
	if len(releaseDate) >= 4 {
		year = releaseDate[:4]
	}
	poster, _ := details["poster_path"].(string)

	return MovieDetails{
		TMDBID:        tmdbID,
		Name:          name,
		Year:          year,
		ReleaseDate:   releaseDate,
		PosterPath:    poster,
		SearchPattern: BuildMovieSearchPattern(name, year),
	}, nil
}

// BuildMovieSearchPattern normalizes a movie title + year for nCore search.
func BuildMovieSearchPattern(name, year string) string {
	p := strings.ToLower(strings.TrimSpace(name + " " + year))
	p = specialCharsFollow.ReplaceAllString(p, "")
	return strings.Join(strings.Fields(p), " ")
}

func cachedMovieReleases(name, year string) ([]Torrent, error) {
	key := name + "\x00" + year

	movieReleaseCacheMu.Lock()
	if entry, ok := movieReleaseCache[key]; ok && time.Since(entry.at) < movieReleaseCacheTTL {
		releases := entry.releases
		movieReleaseCacheMu.Unlock()
		return releases, nil
	}
	movieReleaseCacheMu.Unlock()

	releases, err := searchMovieTorrents(name, year)
	if err != nil {
		return nil, err
	}

	movieReleaseCacheMu.Lock()
	if len(movieReleaseCache) > 100 {
		movieReleaseCache = map[string]movieReleaseCacheEntry{}
	}
	movieReleaseCache[key] = movieReleaseCacheEntry{releases: releases, at: time.Now()}
	movieReleaseCacheMu.Unlock()

	return releases, nil
}

func searchMovieTorrents(name, year string) ([]Torrent, error) {
	patterns := []string{}
	if p := BuildMovieSearchPattern(name, year); p != "" {
		patterns = append(patterns, p)
	}
	if p := BuildMovieSearchPattern(name, ""); p != "" && (len(patterns) == 0 || p != patterns[0]) {
		patterns = append(patterns, p)
	}

	seen := map[string]bool{}
	var all []Torrent
	var firstErr error

	for _, pattern := range patterns {
		found, err := searchNCoreAllPages(pattern, 3)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, t := range found {
			if seen[t.ID] {
				continue
			}
			if !TitleLooksLikeContent(t.Title, name) {
				continue
			}
			if !movieYearMatches(t.Title, name, year) {
				continue
			}
			seen[t.ID] = true
			all = append(all, t)
		}
	}

	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

// searchNCoreAllPages searches nCore up to maxPages and dedupes torrents.
func searchNCoreAllPages(pattern string, maxPages int) ([]Torrent, error) {
	var all []Torrent
	seen := map[string]bool{}

	for page := 1; page <= maxPages; page++ {
		res, err := SearchNCoreFull(SearchRequest{
			Pattern:   pattern,
			Type:      "all_own",
			Where:     "name",
			SortBy:    "seeders",
			SortOrder: "DESC",
			Page:      page,
		})
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if len(res.Torrents) == 0 {
			break
		}
		for _, t := range res.Torrents {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			all = append(all, t)
		}
		if res.NumOfPages > 0 && page >= res.NumOfPages {
			break
		}
	}
	return all, nil
}

func buildMovieResolutions(releases []Torrent) []MovieResolution {
	counts := map[string]int{}
	bestSeeders := map[string]int{}

	for _, t := range releases {
		q := DetectResolution(t.Title)
		if q == "" {
			continue
		}
		counts[q]++
		if t.Seeders > bestSeeders[q] {
			bestSeeders[q] = t.Seeders
		}
	}

	out := make([]MovieResolution, 0, len(counts))
	for _, q := range movieResolutionOrder {
		if counts[q] == 0 {
			continue
		}
		out = append(out, MovieResolution{
			Quality:  q,
			Releases: counts[q],
			Seeders:  bestSeeders[q],
		})
	}

	// Some releases omit the resolution tag entirely. Offer a single Auto
	// choice instead of pretending nothing is available.
	if len(out) == 0 && len(releases) > 0 {
		best := 0
		for _, t := range releases {
			if t.Seeders > best {
				best = t.Seeders
			}
		}
		out = append(out, MovieResolution{
			Quality:  "Auto",
			Releases: len(releases),
			Seeders:  best,
		})
	}
	return out
}

func bestAvailableQuality(releases []Torrent) string {
	for _, q := range movieDefaultResolutionOrder {
		if _, ok := bestMovieRelease(releases, q); ok {
			return q
		}
	}
	return ""
}

func bestMovieRelease(releases []Torrent, quality string) (Torrent, bool) {
	// Prefer clean releases over cams when both exist.
	best, found := bestMovieReleaseMatching(releases, quality, false)
	if found {
		return best, true
	}
	return bestMovieReleaseMatching(releases, quality, true)
}

func bestMovieReleaseMatching(releases []Torrent, quality string, allowCam bool) (Torrent, bool) {
	var best Torrent
	found := false

	for _, t := range releases {
		if quality != "Auto" && DetectResolution(t.Title) != quality {
			continue
		}
		if !allowCam && isCamRelease(t.Title) {
			continue
		}
		if !found || t.Seeders > best.Seeders ||
			(t.Seeders == best.Seeders && t.Date > best.Date) {
			best = t
			found = true
		}
	}
	return best, found
}

// DetectResolution returns the resolution tag found in a release title.
// "SD" is used for legacy releases (XviD/DVDRip) without a resolution tag.
func DetectResolution(title string) string {
	t := strings.ToLower(title)

	switch {
	case strings.Contains(t, "2160p") || strings.Contains(t, "4k") || strings.Contains(t, "uhd"):
		return "2160p"
	case strings.Contains(t, "1080p") || strings.Contains(t, "1080i") ||
		strings.Contains(t, "fullhd") || strings.Contains(t, "full.hd") || strings.Contains(t, "full hd"):
		return "1080p"
	case strings.Contains(t, "720p"):
		return "720p"
	case strings.Contains(t, "480p") || strings.Contains(t, "576p"):
		return "480p"
	case strings.Contains(t, "xvid") || strings.Contains(t, "dvdrip") ||
		strings.Contains(t, "dvd") || strings.Contains(t, "sdtv"):
		return "SD"
	default:
		return ""
	}
}

func isCamRelease(title string) bool {
	t := strings.ToLower(title)
	for _, marker := range []string{
		"hdcam", "camrip", "telesync", "telecine", "dvdscr", "screener",
		".cam.", ".ts.", ".tc.",
	} {
		if strings.Contains(t, marker) {
			return true
		}
	}
	return false
}

// movieYearMatches guards against remakes when searching without a year.
// If the title has years but none match, reject it — unless the movie name
// itself contains that number (Blade Runner 2049, 1917, 2012, …).
func movieYearMatches(title, name, year string) bool {
	if year == "" {
		return true
	}
	titleYears := movieYearRe.FindAllString(title, -1)
	if len(titleYears) == 0 {
		return true
	}
	for _, y := range titleYears {
		if y == year {
			return true
		}
	}
	for _, nameYear := range movieYearRe.FindAllString(name, -1) {
		for _, titleYear := range titleYears {
			if nameYear == titleYear {
				return true
			}
		}
	}
	return false
}

// MovieFolderName is the per-movie folder under qBittorrent's default save path.
func MovieFolderName(name, year string) string {
	base := sanitizeFolderName(name)
	if year != "" {
		return base + "." + year
	}
	return base
}
