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
	Id int64
	Type ChatType
	Title string
	Description sql.NullString
	OwnerId int64
	AvatarUrl sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MessageModel struct {
	Id int64
	ChatId int64
	SenderId int64
	Content string
	StickerId sql.NullInt64
	Edited bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime
}