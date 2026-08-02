package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/database"
	"ncore-tmdb/internal/service"
)

var specialCharsRegex = regexp.MustCompile(`[,:;!@#$%^&*()+=\[\]{}|\\/"'<>?~` + "`" + `]+`)

// ncoreSearchParams is the exact query used to fetch torrents for a TMDB item.
// Shared with the widget so "Open search result" can reopen the same list in /ncore.
type ncoreSearchParams struct {
	Pattern   string `json:"pattern"`
	Type      string `json:"type"`
	Where     string `json:"where"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Page      int    `json:"page"`
}

func buildNcoreSearchParams(name, releaseDate string) ncoreSearchParams {
	year := ""
	if len(releaseDate) >= 4 {
		year = releaseDate[:4]
	}
	pattern := strings.ToLower(fmt.Sprintf("%s %s", name, year))
	pattern = specialCharsRegex.ReplaceAllString(pattern, "")
	pattern = strings.Join(strings.Fields(pattern), " ")

	return ncoreSearchParams{
		Pattern:   pattern,
		Type:      "all_own",
		Where:     "name",
		SortBy:    "seeders",
		SortOrder: "DESC",
		Page:      1,
	}
}

func fetch(c *gin.Context) {
	idStr := c.Param("tmdbID")
	tmdbID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	contentType := ""
	switch {
	case strings.HasPrefix(c.Request.URL.Path, "/api/movie/"):
		contentType = "movie"
	case strings.HasPrefix(c.Request.URL.Path, "/api/tv/"):
		contentType = "tv"
	default:
		contentType = c.Param("type")
	}
	if contentType != "movie" && contentType != "tv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}

	// Check if content is already fetched (cached)
	existingContent, err := database.GetContent(tmdbID, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check content cache"})
		return
	}

	if existingContent != nil && existingContent.Progress == "FETCHED" {
		// Serve from cache
		metadataStr, err := database.GetContentKV(tmdbID, contentType, "metadata")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cached metadata"})
			return
		}

		var details map[string]any
		if metadataStr != "" {
			json.Unmarshal([]byte(metadataStr), &details)
		}

		torrents, err := database.GetTorrentsByContent(tmdbID, contentType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cached torrents"})
			return
		}

		search := buildNcoreSearchParams(existingContent.Name, existingContent.ReleaseDate)

		c.JSON(http.StatusOK, gin.H{
			"tmdbID":   tmdbID,
			"type":     contentType,
			"metadata": details,
			"torrents": torrents,
			"search":   search,
			"cached":   true,
		})
		return
	}

	// 1. Fetch metadata from TMDB APIs if not cached
	var details map[string]any

	switch contentType {
		case "movie":
			details, err = service.GetMovieDetailsTMDB(tmdbID)
		case "tv":
			details, err = service.GetTVDetailsTMDB(tmdbID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch from TMDB"})
		return
	}

	var name, releaseDate string
	if contentType == "movie" {
		name = fmt.Sprintf("%v", details["title"])
		releaseDate = fmt.Sprintf("%v", details["release_date"])
	} else {
		name = fmt.Sprintf("%v", details["name"])
		releaseDate = fmt.Sprintf("%v", details["first_air_date"])
	}

	progress := "FETCHED"
	if existingContent != nil {
		progress = existingContent.Progress
	}

	content := &database.Content{
		TMDBID:      tmdbID,
		Type:        contentType,
		Name:        name,
		ReleaseDate: releaseDate,
		Progress:    progress,
	}

	err = database.InsertContent(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert content"})
		return
	}

	for key, value := range details {
		switch value.(type) {
		case string, float64, int, bool, nil:
			strValue := fmt.Sprintf("%v", value)
			database.SetContentKV(tmdbID, contentType, key, strValue)
		default:
		}
	}

	// Store full metadata as JSON for cache
	metadataJSON, _ := json.Marshal(details)
	database.SetContentKV(tmdbID, contentType, "metadata", string(metadataJSON))

	search := buildNcoreSearchParams(name, releaseDate)

	// 2. Fetch from multiple torrent providers and write to DB
	// Add your new providers to this list as you build them in the service package
	providers := []string{"NCORE", "OTHER_PROVIDER"}

	for _, provider := range providers {
		var fetchedTorrents []service.Torrent // Assuming your service package returns a standard Torrent slice
		var searchErr error

		switch provider {
		case "NCORE":
			fetchedTorrents, searchErr = service.SearchNCore(service.SearchRequest{
				Pattern:   search.Pattern,
				Type:      search.Type,
				Where:     search.Where,
				SortBy:    search.SortBy,
				SortOrder: search.SortOrder,
				Page:      search.Page,
			})
		case "OTHER_PROVIDER":
			// fetchedTorrents, searchErr = service.SearchOtherProvider(search.Pattern)
		}

		if searchErr != nil {
			fmt.Printf("%s search failed: %v\n", provider, searchErr)
			continue // Skip to the next provider if one fails
		}

		// Insert fetched torrents into the database
		for _, t := range fetchedTorrents {
			torrent := &database.Torrent{
				ID:          t.ID,
				Title:       t.Title,
				Key:         t.Key,
				Type:        t.Type,
				Date:        t.Date,
				Seeders:     t.Seeders,
				Leechers:    t.Leechers,
				DownloadURL: t.DownloadURL,
				Provider:    provider, // Dynamically tag the provider
				TMDBID:      tmdbID,
				ContentType: contentType,
			}
			if err := database.InsertTorrent(torrent); err != nil {
				fmt.Printf("Failed to insert torrent %s from %s: %v\n", t.ID, provider, err)
			}
		}
	}

	// 3. Retrieve the combined torrents from the DB to serve to the user
	dbTorrents, err := database.GetTorrentsByContent(tmdbID, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch aggregated torrents from database"})
		return
	}

	// Fetch active downloads from qbit to show progress
	qbitTorrents, _ := service.GetTorrentsStatus()
	downloadingID, _ := database.GetContentKV(tmdbID, contentType, "downloading_torrent_id")

	var activeDownload map[string]any
	if downloadingID != "" {
		// Try to find the torrent in qbit by name/hash matching if possible,
		// but since we don't store hash yet, we'll look for similar name or just return the first one if we only expect one.
		// For now, let's just find if any torrent in qbit matches any of our known torrent IDs in the title (filename was ID.torrent)
		for _, qt := range qbitTorrents {
			if strings.Contains(qt.Name, downloadingID) {
				activeDownload = map[string]any{
					"id":       downloadingID,
					"progress": qt.Progress * 100,
					"status":   qt.Status,
				}
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tmdbID":         tmdbID,
		"type":           contentType,
		"metadata":       details,
		"torrents":       dbTorrents,
		"search":         search,
		"activeDownload": activeDownload,
	})
}
