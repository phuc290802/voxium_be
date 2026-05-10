package main

import (
	"log"
	"os"

	"message-service/database"
	"message-service/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	database.Connect()

	router := gin.Default()

	// Setup CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Println("Message Service running on port", port)
	router.Run(":" + port)
}
