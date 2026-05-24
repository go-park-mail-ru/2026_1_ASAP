package chat

import "time"

type StickerPack struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Slug         *string
	Title        *string
	ThumbnailURL *string
	Name         string
	Id           int64
	SortOrder    int
	Stickers     []Sticker
}

type Sticker struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Slug      *string
	Emoji     *string
	FileURL   string
	Id        int64
	PackID    int64
	Width     *int
	Height    *int
	SortOrder int
}
