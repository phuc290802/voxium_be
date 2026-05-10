package routes

import (
	"message-service/handlers"
	"message-service/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	router.GET("/health", handlers.HealthCheck)
	
	api := router.Group("/api")
	{
		// Auth
		api.POST("/auth/register", handlers.Register)
		api.POST("/auth/login", handlers.Login)

		// Messages (Unprotected for internal ws-gateway call)
		api.POST("/messages", handlers.CreateMessage)
		api.GET("/rooms/:roomId/check-membership", handlers.CheckMembership)    // For ws-gateway
		api.GET("/rooms/:roomId/validate-invite", handlers.ValidateInvite)      // No auth - preview before join

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/rooms", handlers.CreateRoom)
			protected.GET("/rooms", handlers.GetRooms)
			protected.POST("/rooms/:roomId/join", handlers.JoinRoom) // Optional fallback
			protected.POST("/rooms/join-by-code", handlers.JoinByCode)
			protected.POST("/rooms/join-by-link", handlers.JoinByLink)
			protected.GET("/rooms/:roomId/members", handlers.GetMembers)
			protected.DELETE("/rooms/:roomId/members/:userId", handlers.RemoveMember)
			protected.DELETE("/rooms/:roomId", handlers.DeleteRoom)
			protected.POST("/rooms/:roomId/reset-invite", handlers.ResetInvite)

			protected.GET("/messages", handlers.GetMessages)
			protected.PUT("/messages/:id/read", handlers.MarkRead)
			protected.GET("/messages/unread", handlers.GetUnreadMessages)
		}
	}
}
