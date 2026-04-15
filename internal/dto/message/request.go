package message

type RequestSendMessage struct {
	Text   string `json:"text"`
	ChatID int64  `json:"chat_id"`
}

type RequestGetMessages struct {
	BeforeID *int64 `json:"before_id"`
	ChatID   int64  `json:"chat_id"`
	Limit    int    `json:"limit"`
}
