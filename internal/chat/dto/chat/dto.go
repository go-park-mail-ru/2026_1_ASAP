package dto

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type MessageDTO struct {
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00+09:00"`
	Text      string    `json:"text" example:"Hello, my name is Artem"`
	SenderId  int64     `json:"sender_id"`
}

type ChatInformationDTO struct {
	Avatar      *string    `json:"avatar"`
	Description *string    `json:"description"`
	Title       string     `json:"title" example:"Chat Title"`
	ChatType    ChatType   `json:"chat_type" example:"dialog"`
	LastMessage MessageDTO `json:"last_message"`
	ID          int64      `json:"id"`
	OwnerID     int64      `json:"owner_id"`
}

type ChatCreate struct {
	Title     string   `json:"title" example:"Chat Title"`
	Type      ChatType `json:"type" example:"Dialog"`
	MembersID []int64  `json:"members_id"`
	ID        int64    `json:"id"`
}

type RequestUpdateAvatar struct {
	File *media.FileInput
}

type RequestUpdateTitle struct {
	Title string `json:"title"`
}

type RequestUpdateDescription struct {
	Description string `json:"description"`
}

type RequestAddMember struct {
	MembersId []int64 `json:"members_id"`
}

type RequestJoinChannel struct {
	UserId int64 `json:"user_id"`
	ChatId int64 `json:"chat_id"`
}

type RequestDeleteMember struct {
	MemberId int64 `json:"member_id"`
}

type ResponseGetChatMembers struct {
	MembersId []int64 `json:"members_id"`
}
