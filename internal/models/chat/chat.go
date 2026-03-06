package models

import (
	"time"
	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeDialog ChatType="dialog"
	ChatTypeGroup ChatType="group"
	ChatTypeChannel ChatType="channel"
)

type Chat struct {
	ID uuid.UUID `json:"id"`
	Type ChatType `json:"type"`
	Title string `json:"title"`
	MembersID []uuid.UUID `json:"members_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        uuid.UUID `json:"-"`
	ChatID    uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"sender"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

