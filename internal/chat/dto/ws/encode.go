package ws

import (
	"encoding/json"
	"time"

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

func EncodeMessageRead(m *dto.ResponseMarkRead) ([]byte, error) {
	return Encode(MessageRead, m)
}

func EncodeMessageEdit(m *dto.ResponseEditMessage) ([]byte, error) {
	return Encode(MessageUpdate, m)
}

func EncodeMessageDelete(m *dto.ResponseClearMessage) ([]byte, error) {
	return Encode(MessageClear, m)
}

func EncodeVoiceTranscript(m *dto.ResponseVoiceTranscript) ([]byte, error) {
	return Encode(MessageVoiceTranscript, m)
}

func EncodeChatNew(c *chatdto.ChatInformationDTO) ([]byte, error) {
	return Encode(ChatNew, c)
}

type ChatDeletedPayload struct {
	ID int64 `json:"id"`
}

func EncodeChatDeleted(chatID int64) ([]byte, error) {
	return Encode(ChatDeleted, ChatDeletedPayload{ID: chatID})
}

type ChatUpdatedAvatarPayload struct {
	ChatID    int64  `json:"chat_id"`
	AvatarURL string `json:"avatar_url"`
}

func EncodeChatUpdatedAvatar(chatID int64, avatarURL string) ([]byte, error) {
	return Encode(ChatUpdatedAvatar, ChatUpdatedAvatarPayload{
		ChatID:    chatID,
		AvatarURL: avatarURL,
	})
}

type ChatUpdatedTitlePayload struct {
	ChatID int64  `json:"chat_id"`
	Title  string `json:"title"`
}

func EncodeChatUpdatedTitle(chatID int64, title string) ([]byte, error) {
	return Encode(ChatUpdatedTitle, ChatUpdatedTitlePayload{
		ChatID: chatID,
		Title:  title,
	})
}

type ChatUpdatedDescriptionPayload struct {
	ChatID      int64   `json:"chat_id"`
	Description *string `json:"description,omitempty"`
}

func EncodeChatUpdatedDescription(chatID int64, description *string) ([]byte, error) {
	return Encode(ChatUpdatedDescription, ChatUpdatedDescriptionPayload{
		ChatID:      chatID,
		Description: description,
	})
}

type ChatUpdatedMembersPayload struct {
	ChatID           int64   `json:"chat_id"`
	Type             string  `json:"type"`
	UpdatedMembersID []int64 `json:"updated_members_id"`
	Name             string  `json:"name,omitempty"`
}

func EncodeChatUpdatedMembers(chatID int64, changeType string, updatedMemberIDs []int64, name string) ([]byte, error) {
	ids := append([]int64(nil), updatedMemberIDs...)
	return Encode(ChatUpdatedMembers, ChatUpdatedMembersPayload{
		ChatID:           chatID,
		Type:             changeType,
		UpdatedMembersID: ids,
		Name:             name,
	})
}

type PresenceTypingPayload struct {
	ChatID int64 `json:"chat_id"`
	UserID int64 `json:"user_id"`
	Typing bool  `json:"typing"`
}

func EncodePresenceTyping(chatID, userID int64, typing bool) ([]byte, error) {
	return Encode(PresenceTyping, PresenceTypingPayload{
		ChatID: chatID,
		UserID: userID,
		Typing: typing,
	})
}

type PresenceUserPayload struct {
	UserID int64 `json:"user_id"`
}

func EncodePresenceOnline(userID int64) ([]byte, error) {
	return Encode(PresenceOnline, PresenceUserPayload{UserID: userID})
}

func EncodePresenceOffline(userID int64) ([]byte, error) {
	return Encode(PresenceOffline, PresenceUserPayload{UserID: userID})
}

type PresenceLastSeenPayload struct {
	UserID     int64     `json:"user_id"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

func EncodePresenceLastSeen(userID int64, lastSeenAt time.Time) ([]byte, error) {
	return Encode(PresenceLastSeen, PresenceLastSeenPayload{
		UserID:     userID,
		LastSeenAt: lastSeenAt.UTC(),
	})
}

type RequestPresenceTyping struct {
	ChatID int64 `json:"chat_id"`
}
