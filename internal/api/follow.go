package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ncore-tmdb/internal/database"
	"ncore-tmdb/internal/service"
)

// POST /api/follows  { "tmdbId": 123, "quality": "1080p", "skippedSeasons": [1] }
func createFollow(c *gin.Context) {
	var body struct {
		TMDBID         int    `json:"tmdbId" binding:"required"`
		Quality        string `json:"quality"`
		SkippedSeasons []int  `json:"skippedSeasons"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := service.CreateFollow(body.TMDBID, body.Quality, body.SkippedSeasons)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go func(id int64) {
		_, _ = service.CheckFollow(id)
	}(f.ID)

	c.JSON(http.StatusOK, gin.H{"follow": f})
}

// GET /api/follows
func listFollows(c *gin.Context) {
	list, err := database.ListFollows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"follows": list})
}

// GET /api/follows/:tmdbId
func getFollow(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbId"})
		return
	}

	// DB first (fast); TMDB seasons after so a slow TMDB call doesn't block other DB users longer than needed
	f, err := database.GetFollowByTMDB(tmdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []database.FollowItem
	if f != nil {
		items, _ = database.ListFollowItems(f.ID)
	}
	if items == nil {
		items = []database.FollowItem{}
	}

	name, year, _, seasons, tmdbErr := service.GetTVSeasons(tmdbID)
	if tmdbErr != nil {
		seasons = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"follow":  f,
		"items":   items,
		"seasons": seasons,
		"name":    name,
		"year":    year,
	})
}

// DELETE /api/follows/:tmdbId
func deleteFollow(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbId"})
		return
	}
	if err := service.Unfollow(tmdbID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

// POST /api/follows/:tmdbId/check
func checkFollow(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbId"})
		return
	}
	f, err := database.GetFollowByTMDB(tmdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not following"})
		return
	}

	res, err := service.CheckFollow(f.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "result": res})
		return
	}
	f, _ = database.GetFollowByID(f.ID)
	items, _ := database.ListFollowItems(f.ID)
	c.JSON(http.StatusOK, gin.H{"result": res, "follow": f, "items": items})
}

// POST /api/follows/check-all
func checkAllFollows(c *gin.Context) {
	go service.CheckAllFollows()
	c.JSON(http.StatusOK, gin.H{"message": "check started for all active follows"})
}

// PATCH /api/follows/:tmdbId  { "quality", "status", "skippedSeasons" }
func updateFollow(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdbId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tmdbId"})
		return
	}
	f, err := database.GetFollowByTMDB(tmdbID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not following"})
		return
	}
	var body struct {
		Quality        *string `json:"quality"`
		Status         *string `json:"status"`
		SkippedSeasons *[]int  `json:"skippedSeasons"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Quality != nil {
		q := *body.Quality
		if q == "720p" || q == "1080p" {
			f.Quality = q
		}
	}
	if body.Status != nil {
		s := *body.Status
		if s == "active" || s == "paused" {
			f.Status = s
		}
	}
	if body.SkippedSeasons != nil {
		f.SkippedSeasons = *body.SkippedSeasons
	}
	if err := database.UpdateFollow(f); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if body.SkippedSeasons != nil {
		updated, err := service.SetSkippedSeasons(tmdbID, *body.SkippedSeasons)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		f = updated
	} else {
		// Reload counts
		f, _ = database.GetFollowByID(f.ID)
	}
	c.JSON(http.StatusOK, gin.H{"follow": f})
}
