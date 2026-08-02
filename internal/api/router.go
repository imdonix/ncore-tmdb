package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/health", healthCheck)

		// NCore client API (registered before param routes)
		ncore := api.Group("/ncore")
		{
			ncore.POST("/search", ncoreSearch)
			ncore.GET("/torrent/:id", ncoreTorrent)
			ncore.GET("/download/:id", ncoreDownload)
			ncore.POST("/qbit/:id", ncoreQbit)
			ncore.GET("/recommended", ncoreRecommended)
			ncore.GET("/activity", ncoreActivity)
			ncore.GET("/types", ncoreTypes)
		}

		// TMDB-linked endpoints used by the widget
		api.GET("/movie/:tmdbID", fetch)
		api.GET("/tv/:tmdbID", fetch)
		api.GET("/download/:id", download)
		api.GET("/qbit/download/:id", qbitDownload)

		// qBittorrent management for the NCore SPA
		qbit := api.Group("/qbit")
		{
			qbit.GET("/torrents", listQbitTorrents)
			qbit.GET("/torrents/ncore/:id", getQbitByNcore)
			qbit.DELETE("/torrents/:hash", deleteQbitTorrent)
		}

		// Series follow (auto-download episodes)
		follows := api.Group("/follows")
		{
			follows.GET("", listFollows)
			follows.POST("", createFollow)
			follows.POST("/check-all", checkAllFollows)
			follows.GET("/:tmdbId", getFollow)
			follows.PATCH("/:tmdbId", updateFollow)
			follows.DELETE("/:tmdbId", deleteFollow)
			follows.POST("/:tmdbId/check", checkFollow)
		}
	}
}
