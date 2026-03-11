package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	UserID uuid.UUID `json:"user_id"`
	Expire time.Time `json:"expire"`
}

type SessionData struct {
	SessionID string
	Expire    time.Time
}
