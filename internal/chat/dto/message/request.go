package message

type RequestSendMessage struct {
	Text   string `json:"text"`
	ChatID int64  `json:"chat_id"`
}

type RequestSendSticker struct {
	ChatID    int64 `json:"chat_id"`
	StickerID int64 `json:"sticker_id"`
}

type RequestEditMessage struct {
	Text      string `json:"text"`
	MessageID int64  `json:"message_id"`
	ChatID    int64  `json:"chat_id"`
}

type RequestDeleteMessage struct {
	MessageID int64 `json:"message_id"`
	ChatID    int64 `json:"chat_id"`
}

type RequestGetMessages struct {
	BeforeID *int64 `json:"before_id"`
	ChatID   int64  `json:"chat_id"`
	Limit    int    `json:"limit"`
}

type RequestMarkRead struct {
	MessageID int64 `json:"message_id"`
	ChatID    int64 `json:"chat_id"`
}
