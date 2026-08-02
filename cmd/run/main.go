package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"ncore-tmdb/internal/api"
	"ncore-tmdb/internal/database"
	"ncore-tmdb/internal/service"
	"ncore-tmdb/internal/static"

	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := database.Init("data/ncore-tmdb.db"); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.CreateKVTable(); err != nil {
		log.Fatal("Failed to create kv table:", err)
	}
	if err := database.CreateContentTable(); err != nil {
		log.Fatal("Failed to create content table:", err)
	}
	if err := database.CreateTorrentTable(); err != nil {
		log.Fatal("Failed to create torrent table:", err)
	}

	service.InitTMDB()
	service.InitNCore()
	service.InitProxy()
	service.InitQbit()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	r.Use(secure.New(secure.Config{
		AllowedHosts:  []string{"localhost", "tmdb.local", "127.0.0.1"},
		IsDevelopment: true,
	}))

	// --- Widget assets ---
	widgetFS, err := static.WidgetFS()
	if err != nil {
		log.Fatal(err)
	}
	r.StaticFS("/widget", widgetFS)

	// --- NCore SPA under /ncore ---
	// Mount BEFORE the TMDB proxy so it never falls through.
	webappSub, err := static.WebappSub()
	if err != nil {
		log.Fatal(err)
	}
	r.Any("/ncore", spaIndex)
	r.Any("/ncore/*filepath", spaOrAsset(webappSub))

	api.RegisterRoutes(r)

	// TMDB reverse proxy for everything else (incl. /assets, /movie, …)
	// Also registers /sw.js kill-switch.
	service.SetupProxy(r)

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	log.Printf("Server is running on http://localhost%s", addr)
	log.Printf("  TMDB:  http://localhost%s/", addr)
	log.Printf("  NCore: http://localhost%s/ncore", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func spaIndex(c *gin.Context) {
	service.ServeSPA(c)
}

// spaOrAsset serves /ncore/assets/* from the embed FS, otherwise SPA index
// for client-side routes like /ncore/torrent/123.
func spaOrAsset(webapp fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(webapp))

	return func(c *gin.Context) {
		// Gin gives filepath like "/assets/foo.js" or "/torrent/1"
		raw := c.Param("filepath")
		rel := strings.TrimPrefix(raw, "/")
		rel = path.Clean("/" + rel)
		rel = strings.TrimPrefix(rel, "/")

		// Only real files under assets/ are served as static
		if strings.HasPrefix(rel, "assets/") {
			// Verify file exists in embed FS
			if f, err := webapp.Open(rel); err == nil {
				_ = f.Close()
				// Strip /ncore prefix for FileServer: request path should be /assets/...
				c.Request.URL.Path = "/" + rel
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			log.Printf("spa asset missing in embed: %s", rel)
			c.Status(http.StatusNotFound)
			return
		}

		// Client route → SPA shell
		service.ServeSPA(c)
	}
}
