package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"media-manager/internal/database"
	"media-manager/internal/service"
)

var specialCharsRegex = regexp.MustCompile(`[,:;!@#$%^&*()+=\[\]{}|\\/"'<>?~` + "`" + `]+`)

func fetch(c *gin.Context) {
	idStr := c.Param("tmdbID")
	tmdbID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	contentType := c.Param("type")
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

		c.JSON(http.StatusOK, gin.H{
			"tmdbID":   tmdbID,
			"type":     contentType,
			"metadata": details,
			"torrents": torrents,
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

	content := &database.Content{
		TMDBID:      tmdbID,
		Type:        contentType,
		Name:        name,
		ReleaseDate: releaseDate,
		Progress:    "FETCHED",
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

	year := ""
	if len(releaseDate) >= 4 {
		year = releaseDate[:4]
	}

	searchPattern := fmt.Sprintf("%s %s", name, year)
	searchPattern = strings.ToLower(searchPattern)
	searchPattern = specialCharsRegex.ReplaceAllString(searchPattern, "")

	// 2. Fetch from multiple torrent providers and write to DB
	// Add your new providers to this list as you build them in the service package
	providers := []string{"NCORE", "OTHER_PROVIDER"}

	for _, provider := range providers {
		var fetchedTorrents []service.Torrent // Assuming your service package returns a standard Torrent slice
		var searchErr error

		switch provider {
		case "NCORE":
			fetchedTorrents, searchErr = service.SearchNCore(service.SearchRequest{
				Pattern:   searchPattern,
				Type:      "all_own",
				Where:     "name",
				SortBy:    "seeders",
				SortOrder: "desc",
				Page:      1,
			})
		case "OTHER_PROVIDER":
			// fetchedTorrents, searchErr = service.SearchOtherProvider(searchPattern)
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
				Completed:   t.Completed,
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

	c.JSON(http.StatusOK, gin.H{
		"tmdbID":   tmdbID,
		"type":     contentType,
		"metadata": details,
		"torrents": dbTorrents, // Served directly from the DB source of truth
	})
}
