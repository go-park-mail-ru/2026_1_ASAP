package user

import (
	"database/sql"
	"time"
)

type UserModel struct {
	Id           int64
	Login        string
	FirstName    string
	LastName     sql.NullString
	Email        string
	PasswordHash string
	AvatarUrl    sql.NullString
	Bio          sql.NullString
	BirthDate    sql.NullTime
	LastSeenAt   sql.NullTime
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProfileModel struct {
	UserId    int64
	Login     string
	Email     string
	FirstName string
	LastName  sql.NullString
	Avatar    sql.NullString
	Bio       sql.NullString
	BirthDate sql.NullTime
	LastSeen  sql.NullTime
}
