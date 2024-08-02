package types

import (
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

type Credentials struct {
	gorm.Model
	Username string `json:"username"`
	Password string `json:"password"`
}

type Subscriptions struct {
	gorm.Model
	UserID     uint
	FollowerID uint
}

type Message struct {
	gorm.Model
	Sender_id   uint
	Receiver_id uint
	Content     string
}

type Chat struct {
	Username    string
	LastMessage string
}
