package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"message-service/database"
	"message-service/models"
	"message-service/services"

	"github.com/gin-gonic/gin"
)

func CreateRoom(c *gin.Context) {
	userId, _ := c.Get("userId")

	var input struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Look up the user's role from DB
	var user models.User
	if err := database.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	canCreatePublic := user.Role == "super_admin" || user.Role == "leader"

	roomType := input.Type
	if roomType == "" {
		// Default: public for privileged, private for regular users
		if canCreatePublic {
			roomType = "public"
		} else {
			roomType = "private"
		}
	}

	// Enforce: regular users cannot create public rooms
	if roomType == "public" && !canCreatePublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only leaders and super admins can create public rooms"})
		return
	}

	isPrivate := roomType == "private"

	room := models.Room{
		Name:      input.Name,
		Type:      roomType,
		CreatedBy: userId.(uint),
		IsPrivate: isPrivate,
	}

	if isPrivate {
		room.InviteCode = services.GenerateInviteCode()
	}

	if err := database.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room name might already exist"})
		return
	}

	if isPrivate {
		room.InviteLink = services.GenerateInviteLink(room.ID, room.InviteCode)
		database.DB.Save(&room)
	}

	// Add creator to room members with admin role
	roomMember := models.RoomMember{
		UserID: userId.(uint),
		RoomID: room.ID,
		Role:   "admin",
	}
	database.DB.Create(&roomMember)

	c.JSON(http.StatusOK, gin.H{
		"id":         room.ID,
		"name":       room.Name,
		"type":       room.Type,
		"inviteCode": room.InviteCode,
		"inviteLink": room.InviteLink,
	})
}

func GetRooms(c *gin.Context) {
	userId, _ := c.Get("userId")
	var rooms []models.Room
	
	// Get public rooms
	database.DB.Where("type = ?", "public").Find(&rooms)
	
	// Get private rooms user is a member of
	var myRoomIds []uint
	database.DB.Model(&models.RoomMember{}).Where("user_id = ?", userId).Pluck("room_id", &myRoomIds)
	
	var myPrivateRooms []models.Room
	if len(myRoomIds) > 0 {
		database.DB.Where("type = ? AND id IN ?", "private", myRoomIds).Find(&myPrivateRooms)
		rooms = append(rooms, myPrivateRooms...)
	}

	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

func JoinByCode(c *gin.Context) {
	userId, _ := c.Get("userId")
	var input struct {
		InviteCode string `json:"inviteCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var room models.Room
	if err := database.DB.Where("invite_code = ?", input.InviteCode).First(&room).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid invite code"})
		return
	}

	// Add to room_members if not exists
	var member models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userId, room.ID).First(&member).Error; err != nil {
		database.DB.Create(&models.RoomMember{UserID: userId.(uint), RoomID: room.ID, Role: "member"})
	}

	c.JSON(http.StatusOK, gin.H{
		"roomId":   room.ID,
		"roomName": room.Name,
		"type":     room.Type,
	})
}

func JoinByLink(c *gin.Context) {
	userId, _ := c.Get("userId")
	var input struct {
		InviteLink string `json:"inviteLink" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract token from link
	// Example: https://chatty.com/join/eyJyb29... -> eyJyb29...
	// For simplicity, let's assume the frontend sends the token part
	token := input.InviteLink
	
	roomId, code, err := services.DecodeInviteLink(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invite link format"})
		return
	}

	var room models.Room
	if err := database.DB.Where("id = ? AND invite_code = ?", roomId, code).First(&room).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid or expired invite link"})
		return
	}

	// Add to room_members if not exists
	var member models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userId, room.ID).First(&member).Error; err != nil {
		database.DB.Create(&models.RoomMember{UserID: userId.(uint), RoomID: room.ID, Role: "member"})
	}

	c.JSON(http.StatusOK, gin.H{
		"roomId":   room.ID,
		"roomName": room.Name,
		"type":     room.Type,
	})
}

func GetMembers(c *gin.Context) {
	roomIdStr := c.Param("roomId")

	// Verify requester is a member
	requesterId, _ := c.Get("userId")
	var requesterMember models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", requesterId, roomIdStr).First(&requesterMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this room"})
		return
	}

	var members []models.RoomMember
	database.DB.Where("room_id = ?", roomIdStr).Find(&members)

	var response []map[string]interface{}
	for _, m := range members {
		var user models.User
		database.DB.First(&user, m.UserID)
		response = append(response, map[string]interface{}{
			"userId":   user.ID,
			"username": user.Username,
			"role":     m.Role,
			"joinedAt": m.JoinedAt,
		})
	}
	if response == nil {
		response = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"members": response})
}

func RemoveMember(c *gin.Context) {
	userId, _ := c.Get("userId")
	roomId := c.Param("roomId")
	targetUserId := c.Param("userId")

	// Check if requester is admin (by role or by room.CreatedBy)
	var callerMember models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userId, roomId).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can remove members"})
		return
	}
	if callerMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can remove members"})
		return
	}

	if strconv.Itoa(int(userId.(uint))) == targetUserId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove yourself"})
		return
	}

	var room models.Room
	database.DB.First(&room, roomId)
	database.DB.Where("room_id = ? AND user_id = ?", roomId, targetUserId).Delete(&models.RoomMember{})

	// Send WS event
	go func() {
		wsUrl := os.Getenv("WS_GATEWAY_URL")
		if wsUrl == "" { wsUrl = "http://ws-gateway:3004" }
		
		targetId, _ := strconv.Atoi(targetUserId)
		event := map[string]interface{}{
			"type": "kicked_from_room",
			"payload": map[string]interface{}{
				"roomId": roomId,
				"roomName": room.Name,
				"userId": targetId,
				"reason": "removed_by_admin",
			},
		}
		jsonData, _ := json.Marshal(event)
		http.Post(wsUrl+"/broadcast", "application/json", bytes.NewBuffer(jsonData))
	}()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteRoom(c *gin.Context) {
	userId, _ := c.Get("userId")
	roomId := c.Param("roomId")

	var callerMember models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userId, roomId).First(&callerMember).Error; err != nil || callerMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can delete room"})
		return
	}

	var room models.Room
	database.DB.First(&room, roomId)

	// Cascade delete via hooks or manually
	database.DB.Where("room_id = ?", roomId).Delete(&models.RoomMember{})
	database.DB.Where("room_id = ?", roomId).Delete(&models.Message{})
	database.DB.Delete(&room)
	
	// Send WS event
	go func() {
		wsUrl := os.Getenv("WS_GATEWAY_URL")
		if wsUrl == "" { wsUrl = "http://ws-gateway:3004" }
		
		event := map[string]interface{}{
			"type": "room_deleted",
			"payload": map[string]interface{}{
				"roomId": roomId,
				"roomName": room.Name,
			},
		}
		jsonData, _ := json.Marshal(event)
		http.Post(wsUrl+"/broadcast", "application/json", bytes.NewBuffer(jsonData))
	}()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ResetInvite(c *gin.Context) {
	userId, _ := c.Get("userId")
	roomId := c.Param("roomId")

	var callerMember models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userId, roomId).First(&callerMember).Error; err != nil || callerMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can reset invite"})
		return
	}

	var room models.Room
	if err := database.DB.First(&room, roomId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	room.InviteCode = services.GenerateInviteCode()
	room.InviteLink = services.GenerateInviteLink(room.ID, room.InviteCode)
	database.DB.Save(&room)

	c.JSON(http.StatusOK, gin.H{
		"inviteCode": room.InviteCode,
		"inviteLink": room.InviteLink,
	})
}

func CheckMembership(c *gin.Context) {
	userIdStr := c.Query("userId")
	roomIdStr := c.Param("roomId")
	
	var room models.Room
	if id, err := strconv.Atoi(roomIdStr); err == nil {
		err = database.DB.Where("id = ?", id).First(&room).Error
	} else {
		err = database.DB.Where("name = ?", roomIdStr).First(&room).Error
	}
	if room.ID == 0 {
		c.JSON(http.StatusOK, gin.H{"isMember": true}) // fallback for general
		return
	}
	
	if room.Type == "public" {
		c.JSON(http.StatusOK, gin.H{"isMember": true})
		return
	}

	var member models.RoomMember
	if err := database.DB.Where("user_id = ? AND room_id = ?", userIdStr, room.ID).First(&member).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"isMember": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isMember": true})
}

func JoinRoom(c *gin.Context) {
	userId, _ := c.Get("userId")
	roomIdStr := c.Param("roomId")
	
	var room models.Room
	if err := database.DB.First(&room, roomIdStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	roomMember := models.RoomMember{
		UserID: userId.(uint),
		RoomID: room.ID,
		Role:   "member",
	}

	// Ignore error if already joined
	database.DB.Create(&roomMember)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"roomId":  room.ID,
	})
}

// ValidateInvite - No auth required, used to preview room before joining
func ValidateInvite(c *gin.Context) {
	code := c.Query("code")
	link := c.Query("link")

	var room models.Room

	if code != "" {
		if err := database.DB.Where("invite_code = ?", code).First(&room).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid invite code"})
			return
		}
	} else if link != "" {
		// Extract token from full URL or path
		token := link
		if idx := len("/join/"); len(token) > idx && token[:idx] == "/join/" {
			token = token[idx:]
		} else if i := lastIndexOf(token, "/join/"); i >= 0 {
			token = token[i+6:]
		}
		roomId, invCode, err := services.DecodeInviteLink(token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid invite link"})
			return
		}
		if err := database.DB.Where("id = ? AND invite_code = ?", roomId, invCode).First(&room).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Room not found or link expired"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide code or link query param"})
		return
	}

	var memberCount int64
	database.DB.Model(&models.RoomMember{}).Where("room_id = ?", room.ID).Count(&memberCount)

	c.JSON(http.StatusOK, gin.H{
		"valid":       true,
		"roomId":      room.ID,
		"roomName":    room.Name,
		"memberCount": memberCount,
	})
}

func lastIndexOf(s, substr string) int {
	idx := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			idx = i
		}
	}
	return idx
}
