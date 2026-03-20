package dto

import (
	"time"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type MessageDTO struct {
	CreatedAt time.Time       `json:"created_at" example:"2026-01-01T00:00:00+09:00"`
	SenderId    int64           `json:"sender_id"`
	Text      string          `json:"text" example:"Hello, my name is Artem"`
}

type ChatInformationDTO struct {
	LastMessage MessageDTO `json:"last_message"`
	Title       string     `json:"title" example:"Chat Title"`
	ChatType    ChatType   `json:"chat_type" example:"dialog"`
	ID          int64      `json:"id"`
}

type ChatCreate struct {
	Title     string      `json:"title" example:"Chat Title"`
	Type      ChatType    `json:"type" example:"Dialog"`
	MembersID []int64     `json:"members_id"`
	ID        int64       `json:"id"`
}
