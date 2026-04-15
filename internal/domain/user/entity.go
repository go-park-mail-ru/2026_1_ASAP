package user

import (
	"time"
)

type User struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastName     *string
	AvatarUrl    *string
	Bio          *string
	BirthDate    *time.Time
	LastSeenAt   *time.Time
	Login        string
	FirstName    string
	Email        string
	PasswordHash string
	Id           int64
}

func (u User) Username() string {
	if u.LastName != nil && *u.LastName != "" {
		return u.FirstName + " " + *u.LastName
	}
	return u.FirstName
}
