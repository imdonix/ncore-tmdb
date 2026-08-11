package service

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ncore-tmdb/internal/database"
)

var specialCharsFollow = regexp.MustCompile(`[,:;!@#$%^&*()+=\[\]{}|\\/"'<>?~` + "`" + `]+`)

// FollowCheckResult is returned after a check run.
type FollowCheckResult struct {
	FollowID    int64  `json:"followId"`
	TMDBID      int    `json:"tmdbId"`
	Searched    int    `json:"searched"`
	Matched     int    `json:"matched"`
	Scheduled   int    `json:"scheduled"`
	AlreadyHave int    `json:"alreadyHave"`
	Upgraded    int    `json:"upgraded"` // season packs that replaced individual episodes
	Error       string `json:"error,omitempty"`
	LastCheckAt string `json:"lastCheckAt"`
}

var (
	followMu       sync.Mutex
	followChecking = map[int64]bool{}
)

// BuildSeriesSearchPattern normalizes a series name for nCore search.
// Year is intentionally omitted — nCore series titles usually do not include it.
func BuildSeriesSearchPattern(name string) string {
	p := strings.ToLower(strings.TrimSpace(name))
	p = specialCharsFollow.ReplaceAllString(p, "")
	return strings.Join(strings.Fields(p), " ")
}

// CreateFollow starts following a series.
// skippedSeasons are treated as already owned (not downloaded by the checker).
func CreateFollow(tmdbID int, quality string, skippedSeasons []int) (*database.Follow, error) {
	if quality != "720p" && quality != "1080p" {
		quality = "1080p"
	}
	if skippedSeasons == nil {
		skippedSeasons = []int{}
	}

	existing, err := database.GetFollowByTMDB(tmdbID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		changed := false
		if existing.Quality != quality {
			existing.Quality = quality
			changed = true
		}
		if pat := BuildSeriesSearchPattern(existing.Name); pat != "" && pat != existing.SearchPattern {
			existing.SearchPattern = pat
			changed = true
		}
		if !intSlicesEqual(existing.SkippedSeasons, skippedSeasons) {
			existing.SkippedSeasons = skippedSeasons
			changed = true
		}
		if changed {
			_ = database.UpdateFollow(existing)
			_, _, _, seasons, _ := GetTVSeasons(tmdbID)
			_ = applySkippedSeasons(existing, seasons)
		}
		return database.GetFollowByID(existing.ID)
	}

	name, year, poster, seasons, err := GetTVSeasons(tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb: %w", err)
	}

	f := &database.Follow{
		TMDBID:         tmdbID,
		Name:           name,
		Year:           year,
		Quality:        quality,
		SearchPattern:  BuildSeriesSearchPattern(name),
		Status:         "active",
		PosterPath:     poster,
		SkippedSeasons: skippedSeasons,
	}
	if err := database.InsertFollow(f); err != nil {
		return nil, err
	}

	// Seed wanted episodes from TMDB metadata, then apply skips
	for _, s := range seasons {
		if s.EpisodeCount <= 0 {
			continue
		}
		if err := database.EnsureWantedEpisodes(f.ID, s.SeasonNumber, s.EpisodeCount); err != nil {
			log.Printf("follow: ensure episodes S%02d: %v", s.SeasonNumber, err)
		}
	}
	_ = applySkippedSeasons(f, seasons)

	return database.GetFollowByID(f.ID)
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	am := map[int]bool{}
	for _, v := range a {
		am[v] = true
	}
	for _, v := range b {
		if !am[v] {
			return false
		}
	}
	return true
}

// applySkippedSeasons marks skipped seasons as owned; un-skips seasons removed from the list.
func applySkippedSeasons(f *database.Follow, seasons []TVSeasonInfo) error {
	skip := map[int]bool{}
	for _, s := range f.SkippedSeasons {
		skip[s] = true
	}
	epCount := map[int]int{}
	for _, s := range seasons {
		epCount[s.SeasonNumber] = s.EpisodeCount
	}
	// Also cover seasons only present in skip list
	for s := range skip {
		if epCount[s] == 0 {
			epCount[s] = 30 // fallback so we still mark a range
		}
	}
	for season, count := range epCount {
		if skip[season] {
			if err := database.MarkSeasonSkipped(f.ID, season, count); err != nil {
				return err
			}
		} else {
			_ = database.UnskipSeason(f.ID, season)
		}
	}
	return nil
}

// SetSkippedSeasons updates which seasons are ignored by the checker.
func SetSkippedSeasons(tmdbID int, skipped []int) (*database.Follow, error) {
	f, err := database.GetFollowByTMDB(tmdbID)
	if err != nil || f == nil {
		return nil, fmt.Errorf("not following")
	}
	if skipped == nil {
		skipped = []int{}
	}
	f.SkippedSeasons = skipped
	if err := database.UpdateFollow(f); err != nil {
		return nil, err
	}
	_, _, _, seasons, _ := GetTVSeasons(tmdbID)
	_ = applySkippedSeasons(f, seasons)
	return database.GetFollowByID(f.ID)
}

// Unfollow removes a follow and its items.
func Unfollow(tmdbID int) error {
	f, err := database.GetFollowByTMDB(tmdbID)
	if err != nil {
		return err
	}
	if f == nil {
		return nil
	}
	return database.DeleteFollow(f.ID)
}

// CheckFollow searches nCore for missing episodes/season packs and sends matches to qBit.
func CheckFollow(followID int64) (*FollowCheckResult, error) {
	followMu.Lock()
	if followChecking[followID] {
		followMu.Unlock()
		return nil, fmt.Errorf("check already in progress")
	}
	followChecking[followID] = true
	followMu.Unlock()
	defer func() {
		followMu.Lock()
		delete(followChecking, followID)
		followMu.Unlock()
	}()

	f, err := database.GetFollowByID(followID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("follow not found")
	}

	result := &FollowCheckResult{
		FollowID: f.ID,
		TMDBID:   f.TMDBID,
	}

	// Keep search pattern name-only (fix older follows that stored year)
	if fixed := BuildSeriesSearchPattern(f.Name); fixed != "" && fixed != f.SearchPattern {
		f.SearchPattern = fixed
	}

	// Refresh episode list from TMDB (new seasons may appear)
	name, _, _, seasons, tmdbErr := GetTVSeasons(f.TMDBID)
	if tmdbErr != nil {
		log.Printf("follow: tmdb refresh %d: %v", f.TMDBID, tmdbErr)
	} else {
		if name != "" {
			f.Name = name
			f.SearchPattern = BuildSeriesSearchPattern(name)
		}
		for _, s := range seasons {
			if s.EpisodeCount > 0 {
				_ = database.EnsureWantedEpisodes(f.ID, s.SeasonNumber, s.EpisodeCount)
			}
		}
		// Re-apply user skip list after ensuring wanted rows
		_ = applySkippedSeasons(f, seasons)
	}

	items, err := database.ListFollowItems(f.ID)
	if err != nil {
		return nil, err
	}

	skippedSet := map[int]bool{}
	for _, s := range f.SkippedSeasons {
		skippedSet[s] = true
	}

	// Map of completed/covered/skipped episodes
	type key struct{ s, e int }
	done := map[key]bool{}
	seasonPackDone := map[int]bool{}
	for _, it := range items {
		if skippedSet[it.Season] || it.Status == "skipped" {
			if it.Episode == 0 {
				seasonPackDone[it.Season] = true
			}
			if it.Episode > 0 {
				done[key{it.Season, it.Episode}] = true
			}
			continue
		}
		// "cannot_find" is retriable on the next check
		if it.Episode == 0 && (it.Status == "completed" || it.Status == "downloading" || it.Status == "found") {
			seasonPackDone[it.Season] = true
		}
		if it.Episode > 0 && (it.Status == "completed" || it.Status == "downloading" || it.Status == "found" || it.CoveredBy > 0) {
			done[key{it.Season, it.Episode}] = true
		}
	}
	// Season pack covers all episodes of that season
	for _, it := range items {
		if it.Episode > 0 && seasonPackDone[it.Season] {
			done[key{it.Season, it.Episode}] = true
		}
	}

	// Search nCore by series name only (no year)
	torrents, err := searchSeriesTorrents(f.SearchPattern)
	if err != nil {
		f.LastError = err.Error()
		f.LastCheckAt = time.Now().UTC().Format(time.RFC3339)
		_ = database.UpdateFollow(f)
		result.Error = err.Error()
		result.LastCheckAt = f.LastCheckAt
		return result, err
	}
	result.Searched = len(torrents)

	// Filter + parse
	type candidate struct {
		t      Torrent
		season int
		ep     int // 0 = pack
		seed   int
	}
	var packs []candidate
	var episodes []candidate

	for _, t := range torrents {
		if !TitleLooksLikeSeries(t.Title, f.Name) {
			continue
		}
		if !MatchesQuality(t.Title, f.Quality) {
			continue
		}
		parsed, ok := ParseEpisodeFromTitle(t.Title)
		if !ok {
			continue
		}
		result.Matched++
		c := candidate{t: t, season: parsed.Season, ep: parsed.Episode, seed: t.Seeders}
		if parsed.Episode == 0 {
			packs = append(packs, c)
		} else {
			episodes = append(episodes, c)
		}
	}

	// Prefer higher seeders
	sort.Slice(packs, func(i, j int) bool { return packs[i].seed > packs[j].seed })
	sort.Slice(episodes, func(i, j int) bool { return episodes[i].seed > episodes[j].seed })

	// Best pack per season
	bestPack := map[int]candidate{}
	for _, c := range packs {
		if _, ok := bestPack[c.season]; !ok {
			bestPack[c.season] = c
		}
	}
	// Best episode per S/E
	bestEp := map[key]candidate{}
	for _, c := range episodes {
		k := key{c.season, c.ep}
		if _, ok := bestEp[k]; !ok {
			bestEp[k] = c
		}
	}

	// 1) Season packs — fill gaps OR replace already-downloaded individual episodes
	//    When a full-season pack appears after singles were grabbed, delete those
	//    episode torrents (and files) and download the pack instead.
	for season, c := range bestPack {
		if skippedSet[season] {
			continue
		}
		if seasonPackDone[season] {
			result.AlreadyHave++
			continue
		}

		missing := seasonHasMissingEpisodes(season, items, seasons)
		individuals := seasonIndividualReleases(season, items)

		// Nothing to do: no missing eps and no singles to upgrade
		if !missing && len(individuals) == 0 {
			continue
		}

		upgrade := len(individuals) > 0
		if upgrade {
			log.Printf("follow: S%02d pack found — replacing %d individual episode torrent(s)", season, len(individuals))
			removeIndividualEpisodeTorrents(individuals)
			result.Upgraded++
		}

		if err := grabRelease(f, c.t, season, 0); err != nil {
			log.Printf("follow: grab pack S%02d: %v", season, err)
			// Still record the match so the UI can open the nCore torrent
			_ = database.UpsertFollowItem(&database.FollowItem{
				FollowID:       f.ID,
				Season:         season,
				Episode:        0,
				Status:         "found",
				NcoreTorrentID: c.t.ID,
				TorrentTitle:   c.t.Title,
			})
			// If upgrade deleted singles but pack failed to add, leave episodes as cannot_find
			// so the next check can retry; do not leave them as downloading with dead hashes.
			if upgrade {
				for _, it := range individuals {
					_ = database.UpsertFollowItem(&database.FollowItem{
						FollowID: f.ID,
						Season:   it.Season,
						Episode:  it.Episode,
						Status:   "cannot_find",
					})
					done[key{it.Season, it.Episode}] = false
				}
			}
			continue
		}
		result.Scheduled++
		seasonPackDone[season] = true

		// Mark all episodes of this season as covered by the pack
		packItem, _ := database.GetFollowItem(f.ID, season, 0)
		if packItem != nil {
			_ = database.MarkEpisodesCoveredBySeason(f.ID, season, packItem.ID)
			// Clear per-episode torrent refs (pack owns the content now)
			for _, it := range individuals {
				_ = database.UpsertFollowItem(&database.FollowItem{
					FollowID:       f.ID,
					Season:         it.Season,
					Episode:        it.Episode,
					Status:         "completed",
					CoveredBy:      packItem.ID,
					NcoreTorrentID: "",
					TorrentTitle:   "",
					QbitHash:       "",
				})
			}
		}
		for _, it := range items {
			if it.Season == season && it.Episode > 0 {
				done[key{season, it.Episode}] = true
			}
		}
	}

	// 2) Individual episodes still missing
	for k, c := range bestEp {
		if skippedSet[k.s] {
			continue
		}
		if seasonPackDone[k.s] || done[k] {
			result.AlreadyHave++
			continue
		}
		// Only if this episode is in wanted list (or we know season exists)
		it, _ := database.GetFollowItem(f.ID, k.s, k.e)
		if it == nil {
			// Unknown episode — still track as found if we want future seasons only when in TMDB
			// Create wanted row then grab
			_ = database.UpsertFollowItem(&database.FollowItem{
				FollowID: f.ID,
				Season:   k.s,
				Episode:  k.e,
				Status:   "wanted",
			})
		} else if it.Status == "completed" || it.Status == "downloading" || it.Status == "found" {
			result.AlreadyHave++
			continue
		}

		if err := grabRelease(f, c.t, k.s, k.e); err != nil {
			log.Printf("follow: grab S%02dE%02d: %v", k.s, k.e, err)
			// Still link the nCore torrent so the UI can open it
			_ = database.UpsertFollowItem(&database.FollowItem{
				FollowID:       f.ID,
				Season:         k.s,
				Episode:        k.e,
				Status:         "found",
				NcoreTorrentID: c.t.ID,
				TorrentTitle:   c.t.Title,
			})
			result.Matched++
			done[k] = true
			continue
		}
		result.Scheduled++
		done[k] = true
	}

	// Mark remaining wanted / retriable episodes as not found this check
	items, _ = database.ListFollowItems(f.ID)
	for _, it := range items {
		if it.Episode <= 0 {
			continue
		}
		if skippedSet[it.Season] || it.Status == "skipped" {
			continue
		}
		if it.Status == "completed" || it.Status == "downloading" || it.Status == "found" || it.CoveredBy > 0 {
			continue
		}
		if seasonPackDone[it.Season] || done[key{it.Season, it.Episode}] {
			continue
		}
		// Still wanted / failed / cannot_find and no match this run
		_ = database.UpsertFollowItem(&database.FollowItem{
			FollowID: f.ID,
			Season:   it.Season,
			Episode:  it.Episode,
			Status:   "cannot_find",
		})
	}

	f.LastCheckAt = time.Now().UTC().Format(time.RFC3339)
	f.LastError = ""
	_ = database.UpdateFollow(f)
	result.LastCheckAt = f.LastCheckAt

	return result, nil
}

func searchSeriesTorrents(pattern string) ([]Torrent, error) {
	// Prefer Hungarian HD series then general; all_own is broadest
	var all []Torrent
	seen := map[string]bool{}

	// Multiple pages for better coverage
	for page := 1; page <= 3; page++ {
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

// seasonHasMissingEpisodes is true if any episode still needs a release
// (wanted / cannot_find / failed) or TMDB lists episodes we don't have yet.
func seasonHasMissingEpisodes(season int, items []database.FollowItem, seasons []TVSeasonInfo) bool {
	acquired := map[int]bool{} // episode number -> has release
	for _, it := range items {
		if it.Season != season || it.Episode <= 0 {
			continue
		}
		if it.CoveredBy > 0 {
			acquired[it.Episode] = true
			continue
		}
		switch it.Status {
		case "completed", "downloading", "found":
			acquired[it.Episode] = true
		default:
			return true // wanted, cannot_find, failed, …
		}
	}
	for _, s := range seasons {
		if s.SeasonNumber != season || s.EpisodeCount <= 0 {
			continue
		}
		for ep := 1; ep <= s.EpisodeCount; ep++ {
			if !acquired[ep] {
				return true
			}
		}
	}
	return false
}

// seasonIndividualReleases returns episode items that were acquired as singles
// (not covered by a pack) — candidates to delete when upgrading to a season pack.
func seasonIndividualReleases(season int, items []database.FollowItem) []database.FollowItem {
	var out []database.FollowItem
	for _, it := range items {
		if it.Season != season || it.Episode <= 0 {
			continue
		}
		if it.CoveredBy > 0 {
			continue
		}
		switch it.Status {
		case "completed", "downloading", "found":
			if it.NcoreTorrentID != "" || it.QbitHash != "" {
				out = append(out, it)
			}
		}
	}
	return out
}

// removeIndividualEpisodeTorrents deletes each release from qBittorrent with files on disk.
func removeIndividualEpisodeTorrents(items []database.FollowItem) {
	for _, it := range items {
		hash := it.QbitHash
		if hash == "" && it.NcoreTorrentID != "" {
			if qt, err := GetTorrentByNcoreID(it.NcoreTorrentID); err == nil && qt != nil {
				hash = qt.Hash
			}
		}
		if hash == "" {
			// Try matching by torrent title in qBit list
			if it.TorrentTitle != "" {
				if list, err := GetTorrentsStatus(); err == nil {
					for _, qt := range list {
						if strings.EqualFold(qt.Name, it.TorrentTitle) || strings.Contains(qt.Name, it.TorrentTitle) || strings.Contains(it.TorrentTitle, qt.Name) {
							hash = qt.Hash
							break
						}
					}
				}
			}
		}
		if hash == "" {
			log.Printf("follow: no qBit hash to delete for S%02dE%02d (%s)", it.Season, it.Episode, it.TorrentTitle)
			continue
		}
		if err := DeleteTorrent(hash, true); err != nil {
			log.Printf("follow: delete individual S%02dE%02d hash=%s: %v", it.Season, it.Episode, hash, err)
		} else {
			log.Printf("follow: deleted individual S%02dE%02d from qBit (files removed)", it.Season, it.Episode)
		}
	}
}

func grabRelease(f *database.Follow, t Torrent, season, episode int) error {
	data, err := DownloadTorrent(t.ID)
	if err != nil {
		return fmt.Errorf("download torrent: %w", err)
	}
	filename := t.Title
	if filename == "" {
		filename = t.ID
	}
	// Layout under qBit downloads root (Jellyfin-friendly single series tree):
	//   Title/Title.S01/Title.S01E01   (episode)
	//   Title/Title.S01/Title.S01      (season pack)
	seasonDir := FollowSeasonSavePath(f.Name, season)
	episodeName := FollowEpisodeFolderName(f.Name, season, episode)
	if err := AddTorrent(data, filename+".torrent", AddTorrentOpts{
		NcoreID:  t.ID,
		SavePath: seasonDir,
		Rename:   episodeName,
	}); err != nil {
		return fmt.Errorf("qbit: %w", err)
	}

	// Keep nCore id for UI links; try to capture qBit hash for later deletes/upgrades
	qbitHash := ""
	if qt, err := GetTorrentByNcoreID(t.ID); err == nil && qt != nil {
		qbitHash = qt.Hash
	}

	item := &database.FollowItem{
		FollowID:       f.ID,
		Season:         season,
		Episode:        episode,
		Status:         "downloading",
		NcoreTorrentID: t.ID,
		TorrentTitle:   t.Title,
		QbitHash:       qbitHash,
	}
	if err := database.UpsertFollowItem(item); err != nil {
		return err
	}
	return nil
}

// CheckAllFollows runs CheckFollow for every active follow.
func CheckAllFollows() {
	list, err := database.ListActiveFollows()
	if err != nil {
		log.Printf("follow: list active: %v", err)
		return
	}
	log.Printf("follow: checking %d series…", len(list))
	for _, f := range list {
		res, err := CheckFollow(f.ID)
		if err != nil {
			log.Printf("follow: check %s (%d): %v", f.Name, f.TMDBID, err)
			continue
		}
		log.Printf("follow: %s — searched=%d matched=%d scheduled=%d",
			f.Name, res.Searched, res.Matched, res.Scheduled)
	}
}

// StartFollowScheduler runs an hourly check loop.
func StartFollowScheduler() {
	go func() {
		// Small delay after boot so ncore login settles
		time.Sleep(30 * time.Second)
		CheckAllFollows()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			CheckAllFollows()
		}
	}()
	log.Println("follow: hourly scheduler started")
}
