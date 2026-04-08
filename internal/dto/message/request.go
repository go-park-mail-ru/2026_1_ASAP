package message

type RequestSendMessage struct {
	Text string `json:"text"`
}

type RequestGetMessages struct {
	Limit    int    `json:"limit"`
	BeforeID *int64 `json:"before_id"`
}
