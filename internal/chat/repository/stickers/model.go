package stickers

import (
	"database/sql"
	"time"
)

type StickerPackModel struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Slug         sql.NullString
	Title        sql.NullString
	ThumbnailURL sql.NullString
	Name         string
	Id           int64
	SortOrder    int
}

type StickerModel struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Slug      sql.NullString
	Emoji     sql.NullString
	FileURL   string
	Id        int64
	PackID    int64
	Width     sql.NullInt64
	Height    sql.NullInt64
	SortOrder int
}
