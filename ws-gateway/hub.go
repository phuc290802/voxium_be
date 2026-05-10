package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan Event),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.broadcastUsers(client.RoomId)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.broadcastUsers(client.RoomId)
			}

		case event := <-h.broadcast:
			switch event.Type {
			case "send_message":
				payloadBytes, _ := json.Marshal(event.Payload)
				var msgPayload map[string]interface{}
				json.Unmarshal(payloadBytes, &msgPayload)
				
				roomId, _ := msgPayload["roomId"].(string)
				content, _ := msgPayload["content"].(string)
				username, _ := msgPayload["username"].(string)
				userIdFloat, _ := msgPayload["userId"].(float64)

				msgData := map[string]interface{}{
					"roomId":   roomId,
					"content":  content,
					"username": username,
					"userId":   uint(userIdFloat),
				}
				msgJson, _ := json.Marshal(msgData)
				
				messageSvcUrl := os.Getenv("MESSAGE_SERVICE_URL")
				if messageSvcUrl == "" {
					messageSvcUrl = "http://localhost:3001"
				}

				resp, err := http.Post(messageSvcUrl+"/api/messages", "application/json", bytes.NewBuffer(msgJson))
				var savedMsg interface{}
				if err == nil {
					defer resp.Body.Close()
					json.NewDecoder(resp.Body).Decode(&savedMsg)
				} else {
					log.Printf("Failed to save message: %v", err)
					savedMsg = msgData
				}

				outEvent := Event{
					Type:    "new_message",
					Payload: savedMsg,
				}
				h.broadcastToRoom(roomId, outEvent)

			case "typing_start", "typing_stop":
				payloadBytes, _ := json.Marshal(event.Payload)
				var typingPayload map[string]interface{}
				json.Unmarshal(payloadBytes, &typingPayload)
				roomId, _ := typingPayload["roomId"].(string)
				
				isTyping := false
				if event.Type == "typing_start" {
					isTyping = true
				}

				typingPayload["isTyping"] = isTyping

				outEvent := Event{
					Type:    "user_typing",
					Payload: typingPayload,
				}
				h.broadcastToRoom(roomId, outEvent)

			case "mark_read":
				payloadBytes, _ := json.Marshal(event.Payload)
				var readPayload map[string]interface{}
				json.Unmarshal(payloadBytes, &readPayload)
				roomId, _ := readPayload["roomId"].(string)
				
				outEvent := Event{
					Type:    "message_read",
					Payload: readPayload,
				}
				h.broadcastToRoom(roomId, outEvent)

			case "kicked_from_room":
				payloadBytes, _ := json.Marshal(event.Payload)
				var kickPayload map[string]interface{}
				json.Unmarshal(payloadBytes, &kickPayload)
				roomId, _ := kickPayload["roomId"].(string)
				userIdFloat, _ := kickPayload["userId"].(float64)

				for client := range h.clients {
					if client.RoomId == roomId && client.UserID == uint(userIdFloat) {
						select {
						case client.send <- event:
						default:
						}
					}
				}

			case "room_deleted":
				payloadBytes, _ := json.Marshal(event.Payload)
				var delPayload map[string]interface{}
				json.Unmarshal(payloadBytes, &delPayload)
				roomId, _ := delPayload["roomId"].(string)
				h.broadcastToRoom(roomId, event)
			}
		}
	}
}

func (h *Hub) broadcastToRoom(roomId string, event Event) {
	for client := range h.clients {
		if client.RoomId == roomId {
			select {
			case client.send <- event:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

func (h *Hub) broadcastUsers(roomId string) {
	users := []string{}
	for c := range h.clients {
		if c.RoomId == roomId && c.Username != "" {
			users = append(users, c.Username)
		}
	}
	event := Event{
		Type: "user_joined",
		Payload: map[string]interface{}{
			"users": users,
		},
	}
	h.broadcastToRoom(roomId, event)
}
