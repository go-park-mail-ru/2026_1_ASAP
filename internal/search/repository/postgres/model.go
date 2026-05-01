package postgres

import (
	"database/sql"
	"time"
)

type chatSearchRow struct {
	LastMessagePreview sql.NullString
	LastMessageAt      sql.NullTime
	AvatarURL          sql.NullString
	Title              string
	Type               string
	ID                 int64
}

type contactSearchRow struct {
	LastSeen sql.NullTime
	Login    sql.NullString
	Avatar   sql.NullString
	LastName sql.NullString
	UserID   int64
	FName    string
}

type messageSearchRow struct {
	RankDiscard float64
	CreatedAt   time.Time
	Content     sql.NullString
	SenderID    int64
	ChatID      int64
	ID          int64
}
