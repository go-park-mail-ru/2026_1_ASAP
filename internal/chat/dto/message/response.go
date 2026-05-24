package message

import (
	"time"

	stickerdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/sticker"
)

type LastMessageDTO struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	SenderId  int64     `json:"sender_id"`
}

type ResponseSendMessage struct {
	CreatedAt   time.Time              `json:"created_at"`
	Text        string                 `json:"text"`
	ID          int64                  `json:"id"`
	ChatID      int64                  `json:"chat_id"`
	SenderID    int64                  `json:"sender_id"`
	Edited      bool                   `json:"edited"`
	Read        bool                   `json:"read"`
	Attachments []MessageAttachmentDTO `json:"attachments,omitempty"`
	Sticker     *stickerdto.StickerDTO `json:"sticker,omitempty"`
}

type ResponseEditMessage struct {
	CreatedAt         time.Time       `json:"created_at"`
	Text              string          `json:"text"`
	ID                int64           `json:"id"`
	ChatID            int64           `json:"chat_id"`
	SenderID          int64           `json:"sender_id"`
	Edited            bool            `json:"edited"`
	Read              bool            `json:"read"`
	LastMessageEdited bool            `json:"last_message_edited"`
	LastMessage       *LastMessageDTO `json:"last_message,omitempty"`
}

type ResponseClearMessage struct {
	ID                int64           `json:"id"`
	ChatID            int64           `json:"chat_id"`
	SenderID          int64           `json:"sender_id"`
	LastMessageEdited bool            `json:"last_message_edited"`
	LastMessage       *LastMessageDTO `json:"last_message,omitempty"`
}

type ResponseGetMessages struct {
	NextBeforeID *int64       `json:"next_before_id"`
	Messages     []MessageDTO `json:"messages"`
	HasMore      bool         `json:"has_more"`
}

type ResponseMarkRead struct {
	ChatID            int64 `json:"chat_id"`
	ReaderUserID      int64 `json:"reader_user_id"`
	LastReadMessageID int64 `json:"last_read_message_id"`
}
