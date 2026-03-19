package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id           int64
	Username     string
	Email        string
	PasswordHash string
	AvatarUrl    *string
	Bio          *string
	LastSeenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Удалить
type DepricatedUser struct {
	Login        string    `json:"login"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Id           uuid.UUID `json:"id"`
}
