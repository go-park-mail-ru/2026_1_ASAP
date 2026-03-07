package models

import (
	"github.com/google/uuid"
	"time"
)

type Session struct {
	UserID uuid.UUID `json:"user_id"`
	Expire time.Time `json:"expire"`
}