package main

import (
	"testing"
	"time"
)

func TestHubRegistration(t *testing.T) {
	hub := newHub()
	go hub.run()

	client := &Client{
		hub:    hub,
		send:   make(chan Event, 10),
		RoomId: "test-room",
	}

	// Register client
	hub.register <- client

	// Wait for registration
	time.Sleep(10 * time.Millisecond)

	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(hub.clients))
	}

	// Unregister client
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(hub.clients))
	}
}

func TestHubBroadcastToRoom(t *testing.T) {
	hub := newHub()
	go hub.run()

	client1 := &Client{
		hub:    hub,
		send:   make(chan Event, 10),
		RoomId: "room1",
	}
	client2 := &Client{
		hub:    hub,
		send:   make(chan Event, 10),
		RoomId: "room2",
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	// Drain initial user_joined events
	<-client1.send
	<-client2.send

	event := Event{
		Type: "test_event",
		Payload: map[string]string{"msg": "hello"},
	}

	hub.broadcastToRoom("room1", event)

	// Client 1 should receive it
	select {
	case received := <-client1.send:
		if received.Type != "test_event" {
			t.Errorf("Expected test_event, got %s", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 1 timed out waiting for broadcast")
	}

	// Client 2 should NOT receive it
	select {
	case <-client2.send:
		t.Error("Client 2 received message meant for room 1")
	case <-time.After(50 * time.Millisecond):
		// Success
	}
}
