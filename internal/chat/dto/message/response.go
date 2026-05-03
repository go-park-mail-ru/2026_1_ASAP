package message

import "time"

type ResponseSendMessage struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	SenderID  int64     `json:"sender_id"`
	Edited    bool      `json:"edited"`
}

type ResponseDeleteMessage struct {
	ID       int64 `json:"id"`
	ChatID   int64 `json:"chat_id"`
	SenderID int64 `json:"sender_id"`
}

type ResponseGetMessages struct {
	NextBeforeID *int64       `json:"next_before_id"`
	Messages     []MessageDTO `json:"messages"`
	HasMore      bool         `json:"has_more"`
}
