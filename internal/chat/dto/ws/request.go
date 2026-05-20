package ws

import "encoding/json"

type WsRequestType string
type WsErrorCode string
type WsErrorMessage string

const (
	MessageSend     WsRequestType = "message.Send"
	MessageRecv     WsRequestType = "message.Receive"
	MessageEdit     WsRequestType = "message.Edit"
	MessageDelete   WsRequestType = "message.Delete"
	MessageMarkRead WsRequestType = "message.MarkRead"

	PresenceTypingStart WsRequestType = "presence.TypingStart"
	PresenceTypingStop  WsRequestType = "presence.TypingStop"
	PresencePing        WsRequestType = "presence.Ping"
	PresenceBackground  WsRequestType = "presence.Background"
	PresenceForeground  WsRequestType = "presence.Foreground"
)

const (
	ErrCodeInvalidEnvelope          WsErrorCode = "INVALID_ENVELOPE"
	ErrCodeInvalidPayload           WsErrorCode = "INVALID_PAYLOAD"
	ErrCodeUnknownType              WsErrorCode = "UNKNOWN_TYPE"
	ErrCodeEmptyText                WsErrorCode = "EMPTY_TEXT"
	ErrCodeMessageTooLong           WsErrorCode = "MESSAGE_TOO_LONG"
	ErrCodeNotMemberOfChat          WsErrorCode = "NOT_MEMBER_OF_CHAT"
	ErrCodeSendFailed               WsErrorCode = "SEND_FAILED"
	ErrCodeInternal                 WsErrorCode = "INTERNAL_ERROR"
	ErrCodeServerShutdown           WsErrorCode = "SERVER_SHUTDOWN"
	ErrCodeOnlyOwnerCanSendMessaage WsErrorCode = "YOU_CANT_SEND_MESSAGE"
	ErrCodeCantDeleteMessage        WsErrorCode = "YOU_CANT_DELETE_THIS_MESSAGE"
	ErrCodeReadMessageInvalid       WsErrorCode = "READ_MESSAGE_INVALID"
)

const (
	ErrCodeInvalidEnvelopeMsg          WsErrorMessage = "invalid envelope"
	ErrCodeInvalidPayloadMsg           WsErrorMessage = "invalid payload"
	ErrCodeUnknownTypeMsg              WsErrorMessage = "unknown type"
	ErrCodeEmptyTextMsg                WsErrorMessage = "empty text"
	ErrCodeMessageTooLongMsg           WsErrorMessage = "message too long"
	ErrCodeNotMemberOfChatMsg          WsErrorMessage = "not member of chat"
	ErrCodeSendFailedMsg               WsErrorMessage = "send failed"
	ErrCodeInternalMsg                 WsErrorMessage = "internal error"
	ErrCodeServerShutdownMsg           WsErrorMessage = "server shutdown"
	ErrCodeOnlyOwnerCanSendMessaageMsg WsErrorMessage = "only owner of channel can send message"
	ErrCodeCantDeleteMessageMsg        WsErrorMessage = "you cant delete this message"
	ErrCodeReadMessageInvalidMsg       WsErrorMessage = "invalid message for read cursor"
)

type WsRequest struct {
	Type    WsRequestType   `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WsErrorPayload struct {
	Code    WsErrorCode    `json:"code"`
	Message WsErrorMessage `json:"message"`
}
