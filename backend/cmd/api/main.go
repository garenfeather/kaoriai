package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"gpt-tools/backend/internal/server"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := server.NewRouter()
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to start api server: %v", err)
	}
}
