package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/service"
)

// GET /api/movies/:tmdbID
// Returns movie details, available nCore resolutions, and the current download.
func movieOverview(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	overview, err := service.GetMovieOverview(tmdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"movie":       overview.Movie,
		"resolutions": overview.Resolutions,
		"download":    overview.Download,
		"search":      buildNcoreSearchParams(overview.Movie.Name, overview.Movie.ReleaseDate),
		"searchError": overview.SearchError,
	})
}

// POST /api/movies/:tmdbID/download  { "quality": "1080p" }
// Picks the best release for the resolution and sends it to qBittorrent.
func movieDownloadStart(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	var body struct {
		Quality string `json:"quality"`
	}
	// Body is optional; an empty quality lets the service pick the best one.
	_ = c.ShouldBindJSON(&body)

	download, err := service.DownloadMovie(tmdbID, body.Quality)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"download": download})
}

// GET /api/movies/:tmdbID/download
// Lightweight polling endpoint used while a movie is downloading.
func movieDownloadStatus(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbID"})
		return
	}

	download, err := service.GetMovieDownloadState(tmdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"download": download})
}
