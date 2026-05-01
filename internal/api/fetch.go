package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"media-manager/internal/database"
	"media-manager/internal/service"
)

func fetch(c *gin.Context) {
	idStr := c.Param("tmdbID")
	tmdbID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	contentType := c.Param("type")
	var details map[string]any

	switch contentType {
	case "movie":
		details, err = service.GetMovieDetailsTMDB(tmdbID)
	case "tv":
		details, err = service.GetTVDetailsTMDB(tmdbID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
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

	// NCore search
	year := ""
	if len(releaseDate) >= 4 {
		year = releaseDate[:4]
	}

	searchPattern := fmt.Sprintf("%s %s", name, year)
	searchPattern = strings.ToLower(searchPattern)

	torrents, err := service.SearchNCore(service.SearchRequest{
		Pattern:   searchPattern,
		Type:      "all_own", // Default to all_own
		Where:     "name",
		SortBy:    "seeders",
		SortOrder: "desc",
		Page:      1,
	})
	if err != nil {
		fmt.Printf("NCore search failed: %v\n", err)
	}

	for _, t := range torrents {
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
			TMDBID:      tmdbID,
			ContentType: contentType,
		}
		if err := database.InsertTorrent(torrent); err != nil {
			fmt.Printf("Failed to insert torrent %s: %v\n", t.ID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tmdbID":   tmdbID,
		"type":     contentType,
		"metadata": details,
		"torrents": torrents,
	})
}
