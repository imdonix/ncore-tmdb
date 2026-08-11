package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/service"
)

// ncoreSearch handles POST /api/ncore/search
func ncoreSearch(c *gin.Context) {
	var req service.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Type == "" {
		req.Type = "all_own"
	}
	if req.Where == "" {
		req.Where = "name"
	}
	if req.SortBy == "" {
		req.SortBy = "seeders"
	}
	if req.SortOrder == "" {
		req.SortOrder = "DESC"
	}

	result, err := service.SearchNCoreFull(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ncoreTorrent handles GET /api/ncore/torrent/:id
func ncoreTorrent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	data, err := service.GetTorrentDetails(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// ncoreDownload handles GET /api/ncore/download/:id — downloads .torrent file
func ncoreDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	data, err := service.DownloadTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := c.Query("name")
	if filename == "" {
		filename = id
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.torrent\"", filename))
	c.Data(http.StatusOK, "application/x-bittorrent", data)
}

// ncoreQbit handles POST /api/ncore/qbit/:id — send torrent to qBittorrent
func ncoreQbit(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	data, err := service.DownloadTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download torrent file: " + err.Error()})
		return
	}

	filename := c.Query("name")
	if filename == "" {
		filename = id
	}

	if err := service.AddTorrent(data, filename+".torrent", service.AddTorrentOpts{NcoreID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to add torrent to qbittorrent: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "scheduled download in qbittorrent", "ncoreId": id})
}

// ncoreRecommended handles GET /api/ncore/recommended
func ncoreRecommended(c *gin.Context) {
	data, err := service.GetRecommended()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// ncoreActivity handles GET /api/ncore/activity
func ncoreActivity(c *gin.Context) {
	data, err := service.GetActivity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// ncoreTypes handles GET /api/ncore/types — category list for the UI
func ncoreTypes(c *gin.Context) {
	c.JSON(http.StatusOK, service.SearchTypes())
}
