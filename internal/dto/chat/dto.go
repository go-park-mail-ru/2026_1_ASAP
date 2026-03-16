package dto

import (
	"time"

	"github.com/google/uuid"

	dtoUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/user"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type MessageDTO struct {
	CreatedAt time.Time       `json:"created_at" example:"2026-01-01T00:00:00+09:00"`
	Sender    dtoUser.UserDTO `json:"sender"`
	Text      string          `json:"text" example:"Hello, my name is Artem"`
}

type ChatInformationDTO struct {
	LastMessage MessageDTO `json:"last_message"`
	Title       string     `json:"title" example:"Chat Title"`
	ChatType    ChatType   `json:"chat_type" example:"dialog"`
	ID          uuid.UUID  `json:"id" example:"00000000-0000-0000-0000-000000000000"`
}

type ChatCreate struct {
	Title     string      `json:"title" example:"Chat Title"`
	Type      ChatType    `json:"type" example:"Dialog"`
	MembersID []uuid.UUID `json:"members_id" example:"00000000-0000-0000-0000-00000000000"`
	ID        uuid.UUID   `json:"id" example:"00000000-0000-0000-0000-000000000000"`
}
