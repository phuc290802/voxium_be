package handlers

import (
	"net/http"
	"os"
	"time"

	"message-service/database"
	"message-service/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Email:    input.Email,
		Username: input.Username,
		Password: string(hashedPassword),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email or Username already exists"})
		return
	}

	token := generateToken(user.ID, user.Username, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
		"token":    token,
	})
}

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token := generateToken(user.ID, user.Username, user.Role)

	// Get user rooms
	var roomMembers []models.RoomMember
	database.DB.Where("user_id = ?", user.ID).Find(&roomMembers)
	roomIds := []uint{}
	for _, rm := range roomMembers {
		roomIds = append(roomIds, rm.RoomID)
	}
	var rooms []models.Room
	if len(roomIds) > 0 {
		database.DB.Where("id IN ?", roomIds).Find(&rooms)
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
		"token":    token,
		"rooms":    rooms,
	})
}

func generateToken(userId uint, username string, role string) string {
	secret := os.Getenv("JWT_SECRET")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   userId,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}
