package ws

type WsResponseStatus string

const (
	Error         WsResponseStatus = "error"
	MessageNew    WsResponseStatus = "message.New"
	MessageGet    WsResponseStatus = "message.Get"
	MessageUpdate WsResponseStatus = "message.Update"
	MessageClear  WsResponseStatus = "message.Clear"
	ChatNew       WsResponseStatus = "chat.New"
	ChatUpdated   WsResponseStatus = "chat.Updated"
	ChatDeleted   WsResponseStatus = "chat.Deleted"
)

type WsResponse[T any] struct {
	Payload T                `json:"payload"`
	Status  WsResponseStatus `json:"type"`
}
