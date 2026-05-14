package messages

import (
	"database/sql"
	"time"
)

type MessageModel struct {
	Id        int64
	ChatId    int64
	SenderId  int64
	Content   string
	StickerId sql.NullInt64
	Edited    bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime
}
