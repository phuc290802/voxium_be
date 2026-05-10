package main

import (
	"log"
	"os"

	"presence-service/handlers"
	"presence-service/middleware"
	"presence-service/redis"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	redis.InitRedis()

	router := gin.Default()

	// Setup CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api/presence")
	{
		api.GET("/users/:roomId", handlers.GetUsersInRoom)
		api.GET("/user/:userId", handlers.GetUserPresence)
		api.POST("/leave-beacon", handlers.LeaveBeacon)
		api.POST("/leave", middleware.AuthMiddleware(), handlers.LeaveRoom)
		api.POST("/logout", middleware.AuthMiddleware(), handlers.Logout)
		api.POST("/heartbeat", middleware.AuthMiddleware(), handlers.Heartbeat)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3003"
	}

	log.Println("Presence Service running on port", port)
	router.Run(":" + port)
}
