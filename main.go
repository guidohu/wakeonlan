package main

import (
	"log"
	"net/http"
	"os"

	"wakeonlan/config"
	"wakeonlan/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	if envFile := os.Getenv("HOSTS_FILE"); envFile != "" {
		config.HostsFile = envFile
	}
	config.LoadHosts()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.NoRoute(gin.WrapH(http.FileServer(http.Dir("./static"))))

	api := r.Group("/api")
	{
		api.GET("/hosts", gin.WrapF(handlers.HandleHosts))
		api.POST("/hosts", gin.WrapF(handlers.HandleHosts))
		api.DELETE("/hosts/:id", gin.WrapF(handlers.HandleHostDelete))
		api.POST("/hosts/:id/wake", gin.WrapF(handlers.HandleHostWake))
		api.GET("/hosts/:id/ping", gin.WrapF(handlers.HandleHostPing))
		api.PUT("/hosts/:id", gin.WrapF(handlers.HandleHostEdit))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on http://0.0.0.0:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
