package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 10 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 10 * 1024 * 1024 // 10 MB for images
)

type Client struct {
	ID       string
	UserID   uint
	Username string
	RoomId   string
	hub      *Hub
	conn     *websocket.Conn
	send     chan Event
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}
		
		switch event.Type {
		case "join_room":
			payloadBytes, _ := json.Marshal(event.Payload)
			var payload map[string]interface{}
			json.Unmarshal(payloadBytes, &payload)
			
			// JWT Verification
			tokenStr, _ := payload["token"].(string)
			if tokenStr != "" {
				secret := os.Getenv("JWT_SECRET")
				
				token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})
				if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
					if userId, ok := claims["userId"].(float64); ok {
						c.UserID = uint(userId)
					}
					if username, ok := claims["username"].(string); ok {
						c.Username = username
					}
				}
			} else {
				// Fallback to phase 1 mode without auth
				if username, ok := payload["username"].(string); ok {
					c.Username = username
				}
			}

			oldRoomId := c.RoomId
			if roomId, ok := payload["roomId"].(string); ok {
				c.RoomId = roomId
			} else if roomIdFloat, ok := payload["roomId"].(float64); ok {
				c.RoomId = fmt.Sprintf("%.0f", roomIdFloat)
			} else {
				c.RoomId = "general"
			}

			c.hub.register <- c
			
			// If room changed, broadcast to old room too
			if oldRoomId != "" && oldRoomId != c.RoomId {
				c.hub.broadcastUsers(oldRoomId)
			}

			// Check membership
			if c.UserID != 0 {
				messageSvcUrl := os.Getenv("MESSAGE_SERVICE_URL")
				if messageSvcUrl == "" { messageSvcUrl = "http://message-service:3001" } // Use internal docker hostname
				resp, err := http.Get(fmt.Sprintf("%s/api/rooms/%s/check-membership?userId=%d", messageSvcUrl, c.RoomId, c.UserID))
				if err == nil {
					defer resp.Body.Close()
					var res struct {
						IsMember bool `json:"isMember"`
					}
					json.NewDecoder(resp.Body).Decode(&res)
					if !res.IsMember {
						c.send <- Event{Type: "error", Payload: map[string]string{"message": "You are not a member of this private room"}}
						return
					}
				}
			}

			// No redundant register here

		case "send_message", "typing_start", "typing_stop", "mark_read":
			// Inject userId if we have it from connection auth
			if c.UserID != 0 {
				payloadBytes, _ := json.Marshal(event.Payload)
				var payload map[string]interface{}
				json.Unmarshal(payloadBytes, &payload)
				payload["userId"] = c.UserID
				payload["username"] = c.Username
				event.Payload = payload
			}
			c.hub.broadcast <- event

		case "leave_room":
			c.hub.unregister <- c
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
