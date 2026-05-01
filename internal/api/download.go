package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"media-manager/internal/database"
	"media-manager/internal/service"
)

func download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	t, err := database.GetTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find torrent in database"})
		return
	}

	data, err := service.DownloadTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download torrent"})
		return
	}

	filename := t.Title
	if filename == "" {
		filename = id
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.torrent\"", filename))
	c.Header("Content-Type", "application/x-bittorrent")
	c.Data(http.StatusOK, "application/x-bittorrent", data)
}
