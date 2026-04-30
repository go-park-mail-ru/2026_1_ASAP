package ws

import (
	"encoding/json"

	chatdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
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

func EncodeMessageEdit(m *dto.ResponseSendMessage) ([]byte, error) {
	return Encode(MessageUpdate, m)
}

func EncodeChatNew(c *chatdto.ChatInformationDTO) ([]byte, error) {
	return Encode(ChatNew, c)
}

func EncodeChatUpdated(c *chatdto.ChatInformationDTO) ([]byte, error) {
	return Encode(ChatUpdated, c)
}

// ChatDeletedPayload уведомление об удалении чата для всех бывших участников.
type ChatDeletedPayload struct {
	ID int64 `json:"id"`
}

func EncodeChatDeleted(chatID int64) ([]byte, error) {
	return Encode(ChatDeleted, ChatDeletedPayload{ID: chatID})
}
