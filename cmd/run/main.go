package main

import (
	"log"
	"os"

	"media-manager/internal/api"
	"media-manager/internal/database"
	"media-manager/internal/proxy"
	"media-manager/internal/tmdb"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := database.Init("media-manager.db"); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.CreateKVTable(); err != nil {
		log.Fatal("Failed to create kv table:", err)
	}

	if err := database.CreateContentTable(); err != nil {
		log.Fatal("Failed to create content table:", err)
	}

	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		log.Fatal("TMDB_API_KEY environment variable required")
	}
	tmdb.Init(tmdbAPIKey)

	r := gin.Default()

	r.Static("/widget", "./widget")
	r.Static("/dashboard", "./dashboard")

	api.SetupAPI(r)
	proxy.SetupProxy(r)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
