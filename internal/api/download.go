package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"media-manager/internal/service"
)

func download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	data, err := service.DownloadTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download torrent"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.torrent", id))
	c.Header("Content-Type", "application/x-bittorrent")
	c.Data(http.StatusOK, "application/x-bittorrent", data)
}
