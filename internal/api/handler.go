package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupAPI(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/health", healthCheck)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
