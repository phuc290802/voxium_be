package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"presence-service/redis"

	"github.com/gin-gonic/gin"
)

func GetUsersInRoom(c *gin.Context) {
	roomId := c.Param("roomId")
	roomKey := fmt.Sprintf("room:%s", roomId)

	// Get all userIds in the room
	userIds, err := redis.Client.SMembers(redis.Ctx, roomKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	users := []map[string]interface{}{}
	for _, userIdStr := range userIds {
		// Check if user is actually online
		onlineKey := fmt.Sprintf("online:%s", userIdStr)
		isOnline, _ := redis.Client.Exists(redis.Ctx, onlineKey).Result()
		
		if isOnline > 0 {
			userHashKey := fmt.Sprintf("user_info:%s", userIdStr)
			userInfo, _ := redis.Client.HGetAll(redis.Ctx, userHashKey).Result()
			
			// If userInfo is missing or roomId doesn't match, this is a ghost entry
			if len(userInfo) == 0 || userInfo["roomId"] != roomId {
				redis.Client.SRem(redis.Ctx, roomKey, userIdStr)
				continue
			}

			// Extra check: lastSeen must be within the last 40 seconds
			lastSeen, err := time.Parse(time.RFC3339, userInfo["lastSeen"])
			if err == nil && time.Since(lastSeen) > 40*time.Second {
				redis.Client.SRem(redis.Ctx, roomKey, userIdStr)
				continue
			}

			uid, _ := strconv.Atoi(userInfo["userId"])
			users = append(users, map[string]interface{}{
				"userId":   uid,
				"username": userInfo["username"],
				"lastSeen": userInfo["lastSeen"],
			})
		} else {
			// Cleanup offline user from room
			redis.Client.SRem(redis.Ctx, roomKey, userIdStr)
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func GetUserPresence(c *gin.Context) {
	userId := c.Param("userId")
	onlineKey := fmt.Sprintf("online:%s", userId)
	
	isOnline, _ := redis.Client.Exists(redis.Ctx, onlineKey).Result()
	
	if isOnline > 0 {
		userHashKey := fmt.Sprintf("user_info:%s", userId)
		userInfo, _ := redis.Client.HGetAll(redis.Ctx, userHashKey).Result()
		c.JSON(http.StatusOK, gin.H{
			"online":   true,
			"lastSeen": userInfo["lastSeen"],
			"roomId":   userInfo["roomId"],
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"online": false,
		})
	}
}
