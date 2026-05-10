package handlers

import (
	"fmt"
	"net/http"
	"time"

	"presence-service/redis"

	"github.com/gin-gonic/gin"
)

func Heartbeat(c *gin.Context) {
	userId, _ := c.Get("userId")
	username, _ := c.Get("username")
	
	var input struct {
		RoomId interface{} `json:"roomId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roomIdStr := ""
	switch v := input.RoomId.(type) {
	case string:
		roomIdStr = v
	case float64:
		roomIdStr = fmt.Sprintf("%.0f", v)
	default:
		roomIdStr = fmt.Sprintf("%v", v)
	}

	// Save to redis
	onlineKey := fmt.Sprintf("online:%v", userId)
	redis.Client.Set(redis.Ctx, onlineKey, "online", 70*time.Second)
	
	// Get old roomId to cleanup if necessary
	userHashKey := fmt.Sprintf("user_info:%v", userId)
	oldRoomId, _ := redis.Client.HGet(redis.Ctx, userHashKey, "roomId").Result()

	// If room changed, remove from old room
	if oldRoomId != "" && oldRoomId != roomIdStr {
		oldRoomKey := fmt.Sprintf("room:%s", oldRoomId)
		redis.Client.SRem(redis.Ctx, oldRoomKey, userId)
	}

	// Add user to NEW room set
	roomKey := fmt.Sprintf("room:%s", roomIdStr)
	redis.Client.SAdd(redis.Ctx, roomKey, userId)

	// Save user info in a hash for easy retrieval in the room
	redis.Client.HSet(redis.Ctx, userHashKey, map[string]interface{}{
		"userId":   userId,
		"username": username,
		"roomId":   roomIdStr,
		"lastSeen": time.Now().Format(time.RFC3339),
	})
	redis.Client.Expire(redis.Ctx, userHashKey, 70*time.Second)

	c.JSON(http.StatusOK, gin.H{
		"status": "online",
		"ttl":    70,
	})
}
