package messages

import (
	"context"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/message"
)

type MessageServiceInterface interface {
	SendMessage(ctx context.Context, userID int64, chatId int64, req *dto.RequestSendMessage) (*dto.ResponseSendMessage, error)
	GetMessagesByChatId(ctx context.Context, userID int64, chatId int64, req *dto.RequestGetMessages) (*dto.ResponseGetMessages, error)
}
