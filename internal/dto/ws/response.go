package ws

// WsResponseStatus значение поля "type" в JSON (исходящий и входящий контракт).
type WsResponseStatus string

const (
	Error      WsResponseStatus = "error"
	MessageNew WsResponseStatus = "message.New"
	MessageGet WsResponseStatus = "message.Get"
)

// WsResponse исходящий фрейм к клиенту WebSocket.
type WsResponse[T any] struct {
	Status  WsResponseStatus `json:"type"`
	Payload T                `json:"payload"`
}
