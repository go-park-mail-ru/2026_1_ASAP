package sticker

type StickerDTO struct {
	Slug    *string `json:"slug,omitempty"`
	Emoji   *string `json:"emoji,omitempty"`
	FileURL string  `json:"file_url"`
	ID      int64   `json:"id"`
	PackID  int64   `json:"pack_id"`
	Width   *int    `json:"width,omitempty"`
	Height  *int    `json:"height,omitempty"`
}

type StickerPackDTO struct {
	Slug         *string      `json:"slug,omitempty"`
	Title        string       `json:"title"`
	ThumbnailURL *string      `json:"thumbnail_url,omitempty"`
	Name         string       `json:"name"`
	ID           int64        `json:"id"`
	Stickers     []StickerDTO `json:"stickers"`
}

type ResponseGetStickerPacks struct {
	Packs []StickerPackDTO `json:"packs"`
}
