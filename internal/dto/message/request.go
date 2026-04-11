package message

type RequestSendMessage struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type RequestGetMessages struct {
	ChatID   int64  `json:"chat_id"`
	Limit    int    `json:"limit"`
	BeforeID *int64 `json:"before_id"`
}
