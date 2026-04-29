package messages

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

const maxMessageRunes = 2000

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=messages.go -destination=mock/messages_mock.go -package=mock
type MessageRepositoryInterface interface {
	CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error)
	GetMessagesByChatId(ctx context.Context, chatId int64, beforeID *int64, limit int) ([]*domain.Message, error)
}

type ChatRepositoryInterface interface {
	IsMember(ctx context.Context, chatId int64, userId int64) (bool, error)
}

type MessageService struct {
	messageRepo MessageRepositoryInterface
	chatRepo    ChatRepositoryInterface
}

func (m MessageService) GetMessagesByChatId(ctx context.Context, userID int64, chatID int64, req *dto.RequestGetMessages) (*dto.ResponseGetMessages, error) {
	if req == nil {
		return nil, errors.New("get messages nil request")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	// Берем limit+1, чтобы понять has_more
	raw, err := m.messageRepo.GetMessagesByChatId(ctx, chatID, req.BeforeID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("messageRepo get messages: %w", err)
	}

	hasMore := len(raw) > limit
	if hasMore {
		raw = raw[:limit]
	}

	items := make([]dto.MessageDTO, 0, len(raw))
	for _, msg := range raw {
		items = append(items, dto.MessageDTO{
			ID:        msg.Id,
			ChatID:    msg.ChatId,
			SenderID:  msg.SenderId,
			Text:      sanitize.Text(msg.Content),
			CreatedAt: msg.CreatedAt,
		})
	}

	var nextBeforeID *int64
	if len(raw) > 0 {
		lastID := raw[len(raw)-1].Id
		nextBeforeID = &lastID
	}

	return &dto.ResponseGetMessages{
		Messages:     items,
		NextBeforeID: nextBeforeID,
		HasMore:      hasMore,
	}, nil
}

func (m MessageService) SendMessage(ctx context.Context, userID int64, chatId int64, req *dto.RequestSendMessage) (*dto.ResponseSendMessage, error) {
	if req == nil {
		return nil, errors.New("send message nil request")
	}

	if req.Text == "" {
		return nil, domain.ErrMessageEmpty
	}

	if utf8.RuneCountInString(req.Text) > maxMessageRunes {
		return nil, domain.ErrMessageTooLong
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatId, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}

	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	message := &domain.Message{
		ChatId:   chatId,
		SenderId: userID,
		Content:  req.Text,
	}

	createdMessage, err := m.messageRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("messageRepo create message: %w", err)
	}

	return &dto.ResponseSendMessage{
		ID:        createdMessage.Id,
		ChatID:    createdMessage.ChatId,
		SenderID:  createdMessage.SenderId,
		Text:      sanitize.Text(createdMessage.Content),
		CreatedAt: createdMessage.CreatedAt,
	}, nil
}

func NewMessageService(messageRepo MessageRepositoryInterface, chatRepo ChatRepositoryInterface) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
	}
}
