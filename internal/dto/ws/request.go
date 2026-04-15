package ws

import "encoding/json"

type WsRequestType string
type WsErrorCode string
type WsErrorMessage string

const (
	MessageSend WsRequestType = "message.Send"
	MessageRecv WsRequestType = "message.Receive"
)

const (
	ErrCodeInvalidEnvelope WsErrorCode = "INVALID_ENVELOPE"
	ErrCodeInvalidPayload  WsErrorCode = "INVALID_PAYLOAD"
	ErrCodeUnknownType     WsErrorCode = "UNKNOWN_TYPE"
	ErrCodeEmptyText       WsErrorCode = "EMPTY_TEXT"
	ErrCodeMessageTooLong  WsErrorCode = "MESSAGE_TOO_LONG"
	ErrCodeNotMemberOfChat WsErrorCode = "NOT_MEMBER_OF_CHAT"
	ErrCodeSendFailed      WsErrorCode = "SEND_FAILED"
	ErrCodeInternal        WsErrorCode = "INTERNAL_ERROR"
	ErrCodeServerShutdown  WsErrorCode = "SERVER_SHUTDOWN"
)

const (
	ErrCodeInvalidEnvelopeMsg WsErrorMessage = "invalid envelope"
	ErrCodeInvalidPayloadMsg  WsErrorMessage = "invalid payload"
	ErrCodeUnknownTypeMsg     WsErrorMessage = "unknown type"
	ErrCodeEmptyTextMsg       WsErrorMessage = "empty text"
	ErrCodeMessageTooLongMsg  WsErrorMessage = "message too long"
	ErrCodeNotMemberOfChatMsg WsErrorMessage = "not member of chat"
	ErrCodeSendFailedMsg      WsErrorMessage = "send failed"
	ErrCodeInternalMsg        WsErrorMessage = "internal error"
	ErrCodeServerShutdownMsg  WsErrorMessage = "server shutdown"
)

type WsRequest struct {
	Type    WsRequestType   `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WsErrorPayload struct {
	Code    WsErrorCode    `json:"code"`
	Message WsErrorMessage `json:"message"`
}
