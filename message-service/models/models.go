package models

import (
	"time"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"unique;not null;size:100" json:"email"`
	Username  string    `gorm:"unique;not null;size:50" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"default:'user';size:20" json:"role"` // "user", "leader", "super_admin"
	CreatedAt time.Time `json:"createdAt"`
}

type Room struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"unique;not null;size:50" json:"name"`
	Type       string    `gorm:"default:'public'" json:"type"`
	CreatedBy  uint      `json:"createdBy"`
	InviteCode string    `gorm:"unique;size:6" json:"inviteCode,omitempty"`
	InviteLink string    `gorm:"unique;size:255" json:"inviteLink,omitempty"`
	IsPrivate  bool      `gorm:"default:false" json:"isPrivate"`
	CreatedAt  time.Time `json:"createdAt"`
}

type RoomInvite struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RoomID     uint      `gorm:"not null" json:"roomId"`
	InviteCode string    `gorm:"not null;size:6;index" json:"inviteCode"`
	CreatedBy  uint      `gorm:"not null" json:"createdBy"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	MaxUses    *int      `gorm:"default:1" json:"maxUses"`
	UsedCount  int       `gorm:"default:0" json:"usedCount"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

type RoomMember struct {
	UserID   uint      `gorm:"primaryKey" json:"userId"`
	RoomID   uint      `gorm:"primaryKey" json:"roomId"`
	Role     string    `gorm:"default:'member';size:10" json:"role"` // "admin" or "member"
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joinedAt"`
}

type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"type:longtext;not null" json:"content"`
	UserID    uint      `gorm:"not null" json:"userId"`
	RoomID    uint      `gorm:"not null" json:"roomId"`
	ReadBy    string    `gorm:"type:json" json:"readBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Optional relations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}
