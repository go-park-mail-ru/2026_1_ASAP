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
	Id int64
	Type ChatType
	Title string
	Description *string
	OwnerId int64
	AvatarUrl *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	Id int64
	ChatId int64
	SenderId int64
	Content string
	StickerId *int64
	Edited bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}