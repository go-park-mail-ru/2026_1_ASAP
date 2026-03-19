package chat

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type Chat struct {
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Type      ChatType    `json:"type"`
	Title     string      `json:"title"`
	MembersID []uuid.UUID `json:"members_id"`
	ID        uuid.UUID   `json:"id"`
}

type Message struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	ID        uuid.UUID `json:"-"`
	ChatID    uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"sender"`
}
