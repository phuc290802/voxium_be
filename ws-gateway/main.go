package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for phase 1
	},
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan Event, 256),
	}
	
	// Wait for join_room to register properly
	go client.writePump()
	go client.readPump()
}

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3004"
	}

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid event", http.StatusBadRequest)
			return
		}
		hub.broadcast <- event
		w.WriteHeader(http.StatusOK)
	})

	log.Println("WebSocket Gateway running on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
