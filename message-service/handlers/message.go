package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"message-service/database"
	"message-service/models"

	"github.com/gin-gonic/gin"
)

func CreateMessage(c *gin.Context) {
	// Might be called internally by ws-gateway without auth, or with auth
	// For Phase 2, ws-gateway can forward userId
	var input struct {
		Username string `json:"username"`
		Content  string `json:"content" binding:"required"`
		RoomId   string `json:"roomId"` // String format from phase 1, need to map to uint
		UserId   uint   `json:"userId"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var room models.Room
	if id, err := strconv.Atoi(input.RoomId); err == nil {
		err = database.DB.Where("id = ?", id).First(&room).Error
	} else {
		err = database.DB.Where("name = ?", input.RoomId).First(&room).Error
	}
	if room.ID == 0 {
		// Create general room if not exists
		if input.RoomId == "general" || input.RoomId == "" {
			room = models.Room{Name: "general", Type: "public"}
			database.DB.Create(&room)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Room not found"})
			return
		}
	}

	userId := input.UserId
	if userId == 0 {
		// Fallback to find user by username for Phase 1 compatibility
		var user models.User
		database.DB.Where("username = ?", input.Username).First(&user)
		userId = user.ID
	}

	message := models.Message{
		Content: input.Content,
		RoomID:  room.ID,
		UserID:  userId,
		ReadBy:  "[]",
	}

	if err := database.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}
	
	// Preload user for response
	database.DB.Preload("User").First(&message, message.ID)

	// Format response to match Phase 1 / Phase 2 expectations
	c.JSON(http.StatusOK, gin.H{
		"id":        message.ID,
		"content":   message.Content,
		"roomId":    room.Name,
		"userId":    message.UserID,
		"username":  message.User.Username,
		"createdAt": message.CreatedAt,
	})
}

func GetMessages(c *gin.Context) {
	roomIdStr := c.Query("roomId")
	if roomIdStr == "" {
		roomIdStr = "general"
	}

	var room models.Room
	if id, err := strconv.Atoi(roomIdStr); err == nil {
		err = database.DB.Where("id = ?", id).First(&room).Error
	} else {
		err = database.DB.Where("name = ?", roomIdStr).First(&room).Error
	}
	if room.ID == 0 {
		c.JSON(http.StatusOK, gin.H{"messages": []interface{}{}, "hasMore": false})
		return
	}

	userId, _ := c.Get("userId")
	if room.Type == "private" && userId != nil {
		var member models.RoomMember
		if err := database.DB.Where("user_id = ? AND room_id = ?", userId, room.ID).First(&member).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this room"})
			return
		}
	}

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	var messages []models.Message
	database.DB.Where("room_id = ?", room.ID).Order("created_at desc").Limit(limit).Preload("User").Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	formattedMessages := []map[string]interface{}{}
	for _, m := range messages {
		formattedMessages = append(formattedMessages, map[string]interface{}{
			"id":        m.ID,
			"content":   m.Content,
			"userId":    m.UserID,
			"username":  m.User.Username,
			"roomId":    room.Name,
			"createdAt": m.CreatedAt,
			"readBy":    m.ReadBy,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": formattedMessages,
		"hasMore":  false,
	})
}

func MarkRead(c *gin.Context) {
	messageIdStr := c.Param("id")
	messageId, err := strconv.Atoi(messageIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message id"})
		return
	}

	var input struct {
		UserId uint `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var message models.Message
	if err := database.DB.First(&message, messageId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	var readBy []uint
	if message.ReadBy != "" {
		json.Unmarshal([]byte(message.ReadBy), &readBy)
	}

	// Add if not exists
	exists := false
	for _, id := range readBy {
		if id == input.UserId {
			exists = true
			break
		}
	}
	if !exists {
		readBy = append(readBy, input.UserId)
		readByBytes, _ := json.Marshal(readBy)
		message.ReadBy = string(readByBytes)
		database.DB.Save(&message)
	}

	c.JSON(http.StatusOK, gin.H{
		"messageId": messageId,
		"readBy":    readBy,
	})
}

func GetUnreadMessages(c *gin.Context) {
	// Implementation simplified
	c.JSON(http.StatusOK, gin.H{"count": 0, "messages": []interface{}{}})
}
