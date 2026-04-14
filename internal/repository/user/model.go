package user

import (
	"database/sql"
	"time"
)

type UserModel struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	BirthDate    sql.NullTime
	LastSeenAt   sql.NullTime
	Login        string
	FirstName    string
	Email        string
	PasswordHash string
	LastName     sql.NullString
	AvatarUrl    sql.NullString
	Bio          sql.NullString
	Id           int64
}

type ProfileModel struct {
	BirthDate sql.NullTime
	LastSeen  sql.NullTime
	Login     string
	Email     string
	FirstName string
	LastName  sql.NullString
	Avatar    sql.NullString
	Bio       sql.NullString
	UserId    int64
}
