package main

import (
	"log"

	"media-manager/internal/api"
	"media-manager/internal/proxy"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Static("/widget", "./widget")
	r.Static("/dashboard", "./dashboard")

	api.SetupAPI(r)
	proxy.SetupProxy(r)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
