package message

import "time"

type MessageDTO struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
}
