package user

import (
	"database/sql"
	"time"
)

type UserModel struct {
	Id           int64
	Username     string
	Email        string
	PasswordHash string
	AvatarUrl    sql.NullString
	Bio          sql.NullString
	LastSeenAt   sql.NullTime
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
