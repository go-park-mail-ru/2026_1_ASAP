package message

import "time"

type ResponseSendMessage struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type ResponseGetMessages struct {
	Messages     []MessageDTO `json:"messages"`
	NextBeforeID *int64       `json:"next_before_id"`
	HasMore      bool         `json:"has_more"`
}
