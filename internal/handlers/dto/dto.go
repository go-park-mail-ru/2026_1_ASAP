package dto

import (
	"github.com/google/uuid"
	"time"
)

type ChatType string

const (
    ChatTypeDialog  ChatType = "dialog"
    ChatTypeGroup   ChatType = "group"
    ChatTypeChannel ChatType = "channel"
)

type MessageDTO struct {
	Sender uuid.UUID `json:"sender"`
	Text string `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatInformationDTO struct {
	ID uuid.UUID `json:"id"`
	Title string `json:"title"`
	LastMessage MessageDTO `json:"last_message"`
}

type ChatCreate struct {
	ID uuid.UUID `json:"id"`
	Title string `json:"title"`
	Type ChatType `json:"type"`
	MembersID []uuid.UUID `json:"members_id"`
}