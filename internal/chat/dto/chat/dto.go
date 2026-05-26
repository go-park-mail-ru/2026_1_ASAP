package dto

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type RequestUpdateAvatar struct {
	File *media.FileInput
}
