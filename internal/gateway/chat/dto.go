package chat

import "time"

type MessageInfoResponse struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	SenderID  int64     `json:"sender_id"`
}

type ChatInfoResponse struct {
	Avatar            *string             `json:"avatar"`
	Description       *string             `json:"description"`
	Type              string              `json:"type"`
	Title             string              `json:"title"`
	LastMessage       MessageInfoResponse `json:"last_message"`
	ID                int64               `json:"id"`
	OwnerID           int64               `json:"owner_id"`
	UnreadCount       int64               `json:"unread_count"`
	LastReadMessageID int64               `json:"last_read_message_id"`
}

type CreateChatRequest struct {
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	MembersID []int64 `json:"members_id"`
}

type UpdateTitleRequest struct {
	Title string `json:"title"`
}

type UpdateDescriptionRequest struct {
	Description string `json:"description"`
}

type AddMembersRequest struct {
	MembersID []int64 `json:"members_id"`
}

type JoinChannelRequest struct {
	UserId int64 `json:"user_id"`
	ChatId int64 `json:"chat_id"`
}

type ChatMembersResponse struct {
	MembersID []int64 `json:"members_id"`
}

type StickerResponse struct {
	Slug    *string `json:"slug,omitempty"`
	Emoji   *string `json:"emoji,omitempty"`
	FileURL string  `json:"file_url"`
	ID      int64   `json:"id"`
	PackID  int64   `json:"pack_id"`
	Width   *int32  `json:"width,omitempty"`
	Height  *int32  `json:"height,omitempty"`
}

type StickerPackResponse struct {
	Slug         *string           `json:"slug,omitempty"`
	ThumbnailURL *string           `json:"thumbnail_url,omitempty"`
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	ID           int64             `json:"id"`
	Stickers     []StickerResponse `json:"stickers"`
}

type StickerPacksResponse struct {
	Packs []StickerPackResponse `json:"packs"`
}
