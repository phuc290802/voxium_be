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
	secret := "presence-test-secret"
	os.Setenv("JWT_SECRET", secret)

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
			name:           "Valid Token",
			authHeader:     "Bearer " + createToken(1, "testuser"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Expired Token",
			authHeader:     "Bearer expired",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(AuthMiddleware())
			r.GET("/ping", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request, _ = http.NewRequest("GET", "/ping", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			} else if tt.name == "Expired Token" {
				// Create an expired token manually
				expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"userId": 1,
					"exp":    time.Now().Add(-time.Hour).Unix(),
				})
				s, _ := expiredToken.SignedString([]byte(secret))
				c.Request.Header.Set("Authorization", "Bearer "+s)
			}

			r.ServeHTTP(w, c.Request)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected %d, got %d", tt.name, tt.expectedStatus, w.Code)
			}
		})
	}
}
