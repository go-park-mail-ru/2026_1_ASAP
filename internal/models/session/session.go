package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	Expire time.Time `json:"expire"`
	UserID uuid.UUID `json:"user_id"`
}
