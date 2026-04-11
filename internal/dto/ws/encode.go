package ws

import (
	"encoding/json"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/message"
)

func Encode[T any](status WsResponseStatus, payload T) ([]byte, error) {
	return json.Marshal(WsResponse[T]{Status: status, Payload: payload})
}

func EncodeError(p WsErrorPayload) ([]byte, error) {
	return Encode(Error, p)
}

func EncodeMessageNew(m *dto.ResponseSendMessage) ([]byte, error) {
	return Encode(MessageNew, m)
}

func EncodeMessageGet(m *dto.ResponseGetMessages) ([]byte, error) {
	return Encode(MessageGet, m)
}
