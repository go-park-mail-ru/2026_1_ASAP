package messages

import (
	"database/sql"
	"time"
)

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
