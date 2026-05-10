package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"message-service/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	
	if dbHost == "" { dbHost = "localhost" }
	if dbUser == "" { dbUser = "chatty" }
	if dbPassword == "" { dbPassword = "ChattyPass123" }
	if dbName == "" { dbName = "chatty_db" }

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbName)
	
	var err error
	
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database, retrying in 2 seconds... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		log.Fatal("Failed to connect to database completely")
	}

	DB.AutoMigrate(&models.User{}, &models.Room{}, &models.RoomMember{}, &models.Message{}, &models.RoomInvite{})
	log.Println("Database connected and migrated")
}
