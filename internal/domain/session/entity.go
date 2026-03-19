package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	SessionID string    `json:"session_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Старье удалить
type DepricatedSession struct {
	Expire time.Time `json:"expire"`
	UserID uuid.UUID `json:"user_id"`
}
