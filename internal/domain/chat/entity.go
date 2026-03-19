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
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      ChatType
	Title     string
	MembersID []uuid.UUID
	ID        uuid.UUID
}

type Message struct {
	CreatedAt time.Time
	Text      string
	ID        uuid.UUID
	ChatID    uuid.UUID
	UserID    uuid.UUID
}
