package api

import (
	"fmt"
	"net/http"
	"strconv"

	"media-manager/internal/database"
	"media-manager/internal/tmdb"

	"github.com/gin-gonic/gin"
)

func SetupAPI(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/:type/:tmdbID", fetchMovie)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func fetchMovie(c *gin.Context) {
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
			details, err = tmdb.GetMovieDetails(tmdbID)
		case "tv":
			details, err = tmdb.GetTVDetails(tmdbID)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
			return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch from TMDB"})
		return
	}

	content := &database.Content{
		TMDBID:      tmdbID,
		Type:        contentType,
		Name:        fmt.Sprintf("%v", details["title"]),
		ReleaseDate: fmt.Sprintf("%v", details["release_date"]),
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

	c.JSON(http.StatusOK, gin.H{
		"tmdbID": tmdbID,
		"type":   contentType,
		"metadata": details,
	})
}
