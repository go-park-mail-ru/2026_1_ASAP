package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	Expire time.Time
	UserID uuid.UUID
}
