package main

import (
	"log"

	"media-manager/internal/api"
	"media-manager/internal/database"
	"media-manager/internal/service"

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

	if err := database.CreateTorrentTable(); err != nil {
		log.Fatal("Failed to create torrent table:", err)
	}

	service.InitTMDB()
	service.InitNCore()
	service.InitProxy()
	service.InitQbit()

	r := gin.Default()

	r.Static("/widget", "./widget")

	api.RegisterRoutes(r)

	service.SetupProxy(r)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
