package chat

import (
	"database/sql"
	"time"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type ChatModel struct {
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Type              ChatType
	Title             string
	Description       sql.NullString
	AvatarUrl         sql.NullString
	Id                int64
	OwnerId           int64
	LastReadMessageID int64
	UnreadCount       int64
}

type MessageModel struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime
	Content   string
	StickerId sql.NullInt64
	Id        int64
	ChatId    int64
	SenderId  int64
	Edited    bool
}
