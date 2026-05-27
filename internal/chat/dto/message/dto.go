package message

import (
	"time"

	stickerdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/sticker"
)

type MessageDTO struct {
	CreatedAt   time.Time              `json:"created_at"`
	Text        string                 `json:"text"`
	ID          int64                  `json:"id"`
	ChatID      int64                  `json:"chat_id"`
	SenderID    int64                  `json:"sender_id"`
	Edited      bool                   `json:"edited"`
	Attachments []MessageAttachmentDTO `json:"attachments,omitempty"`
	Sticker     *stickerdto.StickerDTO `json:"sticker,omitempty"`
	// Read is meaningful for outgoing messages (sender_id == current user): all other chat members have read up to this message id.
	Read bool `json:"read"`
}
