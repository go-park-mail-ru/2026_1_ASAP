package user

import (
	"time"
)

type User struct {
	Id           int64
	Login        string
	FirstName    string
	LastName     *string
	Email        string
	PasswordHash string
	AvatarUrl    *string
	Bio          *string
	BirthDate    *time.Time
	LastSeenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) Username() string {
    if u.LastName != nil && *u.LastName != "" {
        return u.FirstName + " " + *u.LastName
    }
    return u.FirstName
}
