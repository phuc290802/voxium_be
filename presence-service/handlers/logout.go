package handlers

import (
	"fmt"
	"net/http"
	"os"

	"presence-service/redis"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Logout(c *gin.Context) {
	userId, _ := c.Get("userId")
	
	// 1. Delete online status
	onlineKey := fmt.Sprintf("online:%v", userId)
	redis.Client.Del(redis.Ctx, onlineKey)
	
	// 2. Get current roomId to remove from room set
	userHashKey := fmt.Sprintf("user_info:%v", userId)
	roomId, _ := redis.Client.HGet(redis.Ctx, userHashKey, "roomId").Result()
	
	if roomId != "" {
		roomKey := fmt.Sprintf("room:%s", roomId)
		redis.Client.SRem(redis.Ctx, roomKey, userId)
	}
	
	// 3. Delete user info
	redis.Client.Del(redis.Ctx, userHashKey)
	
	c.JSON(http.StatusOK, gin.H{"message": "Logged out and presence cleared"})
}

func LeaveRoom(c *gin.Context) {
	userId, _ := c.Get("userId")
	
	var input struct {
		RoomId interface{} `json:"roomId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roomIdStr := ""
	if v, ok := input.RoomId.(string); ok {
		roomIdStr = v
	} else if v, ok := input.RoomId.(float64); ok {
		roomIdStr = fmt.Sprintf("%.0f", v)
	}

	if roomIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roomId format"})
		return
	}

	// 1. Remove from room set
	roomKey := fmt.Sprintf("room:%s", roomIdStr)
	redis.Client.SRem(redis.Ctx, roomKey, userId)
	
	// 2. Update user info hash
	userHashKey := fmt.Sprintf("user_info:%v", userId)
	redis.Client.HDel(redis.Ctx, userHashKey, "roomId")
	
	c.JSON(http.StatusOK, gin.H{"message": "Left room"})
}

func LeaveBeacon(c *gin.Context) {
	// For beacon, we might get token from query since sendBeacon doesn't support headers well
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	// Manual JWT parse since middleware might not handle query token
	secret := os.Getenv("JWT_SECRET")
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userId := uint(claims["userId"].(float64))
		
		var input struct {
			RoomId interface{} `json:"roomId"`
		}
		c.ShouldBindJSON(&input)

		roomIdStr := ""
		if v, ok := input.RoomId.(string); ok {
			roomIdStr = v
		} else if v, ok := input.RoomId.(float64); ok {
			roomIdStr = fmt.Sprintf("%.0f", v)
		}

		if roomIdStr != "" {
			roomKey := fmt.Sprintf("room:%s", roomIdStr)
			redis.Client.SRem(redis.Ctx, roomKey, userId)
			
			userHashKey := fmt.Sprintf("user_info:%v", userId)
			redis.Client.HDel(redis.Ctx, userHashKey, "roomId")
		}
	}
	
	c.Status(http.StatusNoContent)
}
