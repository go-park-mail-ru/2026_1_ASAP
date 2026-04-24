package user

import (
	"strings"
	"time"
)

type User struct {
	ID           int64
	Login        string
	Email        string
	PasswordHash string
	VKID         *int64
	FirstName    string
	LastName     *string
	AvatarUrl    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) Username() string {
	if u == nil {
		return ""
	}
	if u.FirstName != "" {
		if u.LastName != nil && *u.LastName != "" {
			return strings.TrimSpace(u.FirstName + " " + *u.LastName)
		}
		return u.FirstName
	}
	if u.Login != "" {
		return u.Login
	}
	return ""
}
