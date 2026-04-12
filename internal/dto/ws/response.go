package ws

type WsResponseStatus string

const (
	Error       WsResponseStatus = "error"
	MessageNew  WsResponseStatus = "message.New"
	MessageGet  WsResponseStatus = "message.Get"
	ChatNew     WsResponseStatus = "chat.New"
	ChatUpdated WsResponseStatus = "chat.Updated"
	ChatDeleted WsResponseStatus = "chat.Deleted"
)

type WsResponse[T any] struct {
	Status  WsResponseStatus `json:"type"`
	Payload T                `json:"payload"`
}
