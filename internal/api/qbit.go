package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"media-manager/internal/database"
	"media-manager/internal/service"
)

func qbitDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing torrent id"})
		return
	}

	// 1. Get torrent info from DB
	t, err := database.GetTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find torrent in database"})
		return
	}

	// 2. Download torrent file content
	data, err := service.DownloadTorrent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download torrent file"})
		return
	}

	// 3. Add to qbittorrent
	filename := t.Title
	if filename == "" {
		filename = id
	}

	err = service.AddTorrent(data, fmt.Sprintf("%s.torrent", filename))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to add torrent to qbittorrent: %v", err)})
		return
	}

	// 4. Update content progress
	err = database.UpdateContentProgress(t.TMDBID, t.ContentType, "DOWNLOADING")
	if err != nil {
		fmt.Printf("Warning: failed to update content progress: %v\n", err)
	}

	// Store what torrent is being downloaded - we use the title for qbit matching
	database.SetContentKV(t.TMDBID, t.ContentType, "downloading_torrent_id", filename)

	c.JSON(http.StatusOK, gin.H{"message": "scheduled download in qbittorrent"})
}
