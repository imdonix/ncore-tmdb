package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/database"
	"ncore-tmdb/internal/service"
)

// listQbitTorrents GET /api/qbit/torrents
func listQbitTorrents(c *gin.Context) {
	list, err := service.GetTorrentsStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []service.QbitTorrent{}
	}
	// Enrich missing ncore links via title match (legacy adds without tags)
	for i := range list {
		if list[i].NcoreID == "" {
			if id := database.FindTorrentIDByName(list[i].Name); id != "" {
				list[i].NcoreID = id
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"torrents": list})
}

// getQbitByNcore GET /api/qbit/torrents/ncore/:id
func getQbitByNcore(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing ncore id"})
		return
	}
	t, err := service.GetTorrentByNcoreID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if t == nil {
		c.JSON(http.StatusOK, gin.H{"torrent": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"torrent": t})
}

// deleteQbitTorrent DELETE /api/qbit/torrents/:hash
// Query deleteFiles=true (default true) removes files on disk.
func deleteQbitTorrent(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing hash"})
		return
	}

	deleteFiles := true
	if v := c.Query("deleteFiles"); v != "" {
		deleteFiles = v == "1" || strings.EqualFold(v, "true") || v == "yes"
	}

	if err := service.DeleteTorrent(hash, deleteFiles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "hash": hash, "deleteFiles": deleteFiles})
}
