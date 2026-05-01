package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/health", healthCheck)
		api.GET("/:type/:tmdbID", fetch)
		api.GET("/download/:id", download)
		api.GET("/qbit/download/:id", qbitDownload)
	}
}
