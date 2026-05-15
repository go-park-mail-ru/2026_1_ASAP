package chat

import (
	"time"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type Chat struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description *string
	AvatarUrl   *string
	Type        ChatType
	Title       string
	Id          int64
	OwnerId     int64
}

type Message struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	StickerId *int64
	DeletedAt *time.Time
	Content   string
	Id        int64
	ChatId    int64
	SenderId  int64
	Edited    bool
}
