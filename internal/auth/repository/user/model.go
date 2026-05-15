package user

import (
	"time"
)

type UserModel struct {
	ID           int64
	Login        string
	Email        string
	PasswordHash string
	VKID         *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
