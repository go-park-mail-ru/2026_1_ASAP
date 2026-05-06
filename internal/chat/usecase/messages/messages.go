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
	UpdateMessage(ctx context.Context, message *domain.Message) (*domain.Message, bool, error)
	DeleteMessage(ctx context.Context, message *domain.Message) (*domain.Message, bool, error)
}

type ChatRepositoryInterface interface {
	IsMember(ctx context.Context, chatId int64, userId int64) (bool, error)
	GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error)
	GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error)
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

	chat, err := m.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember && chat.Type != domain.ChatTypeChannel {
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
			Edited:    msg.Edited,
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

	chat, err := m.chatRepo.GetChatByID(ctx, chatId)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat by id: %w", err)
	}

	if chat.Type == domain.ChatTypeChannel {
		if userID != chat.OwnerId {
			return nil, domain.ErrOnlyOwnerCanSendMessaage
		}
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
		Edited:    createdMessage.Edited,
	}, nil
}

func (m MessageService) EditMessage(ctx context.Context, userID, chatID int64, req *dto.RequestEditMessage) (*dto.ResponseEditMessage, error) {
	if req == nil {
		return nil, errors.New("edit message nil request")
	}

	if req.Text == "" {
		return nil, domain.ErrMessageEmpty
	}

	if utf8.RuneCountInString(req.Text) > maxMessageRunes {
		return nil, domain.ErrMessageTooLong
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}

	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	message := &domain.Message{
		Id:       req.MessageID,
		ChatId:   chatID,
		SenderId: userID,
		Content:  req.Text,
	}

	editedMessage, lastMessageEdited, err := m.messageRepo.UpdateMessage(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("messageRepo updated message: %w", err)
	}

	resp := &dto.ResponseEditMessage{
		ID:                editedMessage.Id,
		ChatID:            editedMessage.ChatId,
		SenderID:          editedMessage.SenderId,
		Text:              sanitize.Text(editedMessage.Content),
		CreatedAt:         editedMessage.CreatedAt,
		Edited:            editedMessage.Edited,
		LastMessageEdited: lastMessageEdited,
	}
	if lastMessageEdited {
		resp.LastMessage = &dto.LastMessageDTO{
			SenderId:  editedMessage.SenderId,
			Text:      sanitize.Text(editedMessage.Content),
			CreatedAt: editedMessage.CreatedAt,
		}
	}
	return resp, nil
}

func (m MessageService) DeleteMessage(ctx context.Context, userID, chatID int64, req *dto.RequestDeleteMessage) (*dto.ResponseClearMessage, error) {
	if req == nil {
		return nil, errors.New("delete message nil request")
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}

	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	message := &domain.Message{
		Id:       req.MessageID,
		ChatId:   chatID,
		SenderId: userID,
	}

	deletedMessage, lastMessageEdited, err := m.messageRepo.DeleteMessage(ctx, message)
	if err != nil {
		if errors.Is(err, domain.ErrNoMessage) {
			return nil, domain.ErrNoMessage
		}
		return nil, fmt.Errorf("messageRepo delete message: %w", err)
	}

	resp := &dto.ResponseClearMessage{
		ID:                deletedMessage.Id,
		ChatID:            deletedMessage.ChatId,
		SenderID:          deletedMessage.SenderId,
		LastMessageEdited: lastMessageEdited,
	}
	if lastMessageEdited {
		lm, err := m.chatRepo.GetLastMessageOfChat(ctx, chatID)
		if err == nil && lm != nil {
			resp.LastMessage = &dto.LastMessageDTO{
				SenderId:  lm.SenderId,
				Text:      sanitize.Text(lm.Content),
				CreatedAt: lm.CreatedAt,
			}
		}
	}
	return resp, nil
}

func NewMessageService(messageRepo MessageRepositoryInterface, chatRepo ChatRepositoryInterface) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
	}
}
