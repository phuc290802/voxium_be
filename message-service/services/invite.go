package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateInviteLink(roomID uint, code string) string {
	data := fmt.Sprintf(`{"roomId":%d,"code":"%s"}`, roomID, code)
	encoded := base64.URLEncoding.EncodeToString([]byte(data))
	return "/join/" + encoded
}

func DecodeInviteLink(token string) (roomID uint, code string, err error) {
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, "", err
	}
	var data struct {
		RoomID uint   `json:"roomId"`
		Code   string `json:"code"`
	}
	err = json.Unmarshal(decoded, &data)
	return data.RoomID, data.Code, err
}
