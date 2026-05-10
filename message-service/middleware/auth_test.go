package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	os.Setenv("JWT_SECRET", secret)

	// Helper to create token
	createToken := func(userId uint, username string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"userId":   userId,
			"username": username,
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		s, _ := token.SignedString([]byte(secret))
		return s
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No Auth Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Format",
			authHeader:     "Token mytoken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid Token",
			authHeader:     "Bearer " + createToken(1, "testuser"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(AuthMiddleware())
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request, _ = http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			r.ServeHTTP(w, c.Request)

			if w.Code != tt.expectedStatus {
				t.Errorf("Test '%s' failed: expected status %d, got %d", tt.name, tt.expectedStatus, w.Code)
			}
		})
	}
}
