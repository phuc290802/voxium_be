package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	// Setup Gin to Test mode
	gin.SetMode(gin.TestMode)

	// Create a response recorder
	w := httptest.NewRecorder()
	
	// Create a test context
	c, _ := gin.CreateTestContext(w)

	// Call the handler
	HealthCheck(c)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}

	if response["timestamp"] == nil {
		t.Error("Expected timestamp in response")
	}
}
