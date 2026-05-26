package messages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

const maxMessageRunes = 2000

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=messages.go -destination=mock/messages_mock.go -package=mock
type MessageRepositoryInterface interface {
	CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error)
	CreateMessageWithAttachments(ctx context.Context, message *domain.Message, attachments []domain.MessageAttachment) (*domain.Message, error)
	GetMessagesByChatId(ctx context.Context, chatId int64, beforeID *int64, limit int) ([]*domain.Message, error)
	GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]domain.MessageAttachment, error)
	GetMessageByID(ctx context.Context, chatID, messageID int64) (*domain.Message, error)
	UpdateAttachmentTranscript(ctx context.Context, attachmentID int64, transcript string) (*domain.MessageAttachment, error)
	CanUserAccessAttachment(ctx context.Context, userID int64, objectKey, attachmentRef string) (bool, error)
	UpdateMessage(ctx context.Context, message *domain.Message) (*domain.Message, bool, error)
	DeleteMessage(ctx context.Context, message *domain.Message) (*domain.Message, bool, error)
}

type StickerRepositoryInterface interface {
	GetStickerByID(ctx context.Context, stickerID int64) (*domain.Sticker, error)
	GetStickersByIDs(ctx context.Context, stickerIDs []int64) (map[int64]domain.Sticker, error)
}

type ChatRepositoryInterface interface {
	IsMember(ctx context.Context, chatId int64, userId int64) (bool, error)
	GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error)
	GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error)
	GetMemberLastReads(ctx context.Context, chatID int64) (map[int64]*int64, error)
	UpdateMemberLastReadMessageID(ctx context.Context, chatID, userID, messageID int64) (effectiveLastRead int64, updated bool, err error)
}

type MessageService struct {
	messageRepo         MessageRepositoryInterface
	chatRepo            ChatRepositoryInterface
	mediaRepo           MessageMediaRepositoryInterface
	profileRepo         ProfileContactsInterface
	subscription        SubscriptionChecker
	stickerRepo         StickerRepositoryInterface
	attachmentProxyBase string
}

func NewMessageService(
	messageRepo MessageRepositoryInterface,
	chatRepo ChatRepositoryInterface,
	mediaRepo MessageMediaRepositoryInterface,
	profileRepo ProfileContactsInterface,
	attachmentProxyBase string,
	subscription SubscriptionChecker,
	stickerRepo ...StickerRepositoryInterface,
) *MessageService {
	var stickers StickerRepositoryInterface
	if len(stickerRepo) > 0 {
		stickers = stickerRepo[0]
	}
	return &MessageService{
		messageRepo:         messageRepo,
		chatRepo:            chatRepo,
		mediaRepo:           mediaRepo,
		profileRepo:         profileRepo,
		subscription:        subscription,
		stickerRepo:         stickers,
		attachmentProxyBase: strings.TrimRight(attachmentProxyBase, "/"),
	}
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

	lastReads, err := m.chatRepo.GetMemberLastReads(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo member last reads: %w", err)
	}

	messageIDs := make([]int64, 0, len(raw))
	for _, msg := range raw {
		messageIDs = append(messageIDs, msg.Id)
	}
	attachmentsByMessage, err := m.messageRepo.GetAttachmentsByMessageIDs(ctx, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("messageRepo get attachments: %w", err)
	}
	stickersByID, err := m.stickersByMessage(ctx, raw)
	if err != nil {
		return nil, err
	}

	subscriptionActive, err := m.isSubscriptionActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check subscription: %w", err)
	}

	items := make([]dto.MessageDTO, 0, len(raw))
	for _, msg := range raw {
		items = append(items, dto.MessageDTO{
			ID:          msg.Id,
			ChatID:      msg.ChatId,
			SenderID:    msg.SenderId,
			Text:        sanitize.Text(msg.Content),
			CreatedAt:   msg.CreatedAt,
			Edited:      msg.Edited,
			Read:        outgoingReadByPeers(msg.Id, msg.SenderId, userID, lastReads),
			Attachments: mapAttachmentsToDTOForViewer(attachmentsByMessage[msg.Id], subscriptionActive),
			Sticker:     stickerDTOFromMessage(msg, stickersByID),
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
		Read:      false,
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

	lastReads, lrErr := m.chatRepo.GetMemberLastReads(ctx, chatID)
	if lrErr != nil {
		return nil, fmt.Errorf("chatrepo member last reads: %w", lrErr)
	}
	read := outgoingReadByPeers(editedMessage.Id, editedMessage.SenderId, userID, lastReads)

	resp := &dto.ResponseEditMessage{
		ID:                editedMessage.Id,
		ChatID:            editedMessage.ChatId,
		SenderID:          editedMessage.SenderId,
		Text:              sanitize.Text(editedMessage.Content),
		CreatedAt:         editedMessage.CreatedAt,
		Edited:            editedMessage.Edited,
		Read:              read,
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

func (m MessageService) MarkMessagesRead(ctx context.Context, userID int64, chatID int64, req *dto.RequestMarkRead) (*dto.ResponseMarkRead, error) {
	if req == nil {
		return nil, errors.New("mark read nil request")
	}
	if req.MessageID <= 0 || chatID <= 0 {
		return nil, domain.ErrReadMessageInvalid
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	effective, ok, err := m.chatRepo.UpdateMemberLastReadMessageID(ctx, chatID, userID, req.MessageID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo update last read: %w", err)
	}
	if !ok {
		return nil, domain.ErrReadMessageInvalid
	}

	return &dto.ResponseMarkRead{
		ChatID:            chatID,
		ReaderUserID:      userID,
		LastReadMessageID: effective,
	}, nil
}

// outgoingReadByPeers reports whether every chat member except the viewer has a read cursor >= messageID
// for messages authored by the viewer (read receipts).
func outgoingReadByPeers(messageID, messageSenderID, viewerID int64, lastReads map[int64]*int64) bool {
	if messageSenderID != viewerID {
		return false
	}
	for uid, lr := range lastReads {
		if uid == viewerID {
			continue
		}
		if lr == nil || *lr < messageID {
			return false
		}
	}
	return true
}
