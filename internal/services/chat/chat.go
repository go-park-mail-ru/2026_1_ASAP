package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
)


type ChatRepositoryInterface interface {
	GetAllChatsByUserID(ctx context.Context, id int64) ([]*domain.Chat, error)
	GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error)
	CreateChat(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
	GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error)
	GetLastMessagesOfChats(ctx context.Context, id int64) ([]*domain.Message, error)
	GetChatMembers(ctx context.Context, chatID int64) ([]int64, error)
	AddMember(ctx context.Context, chatID, userID int64) (error)
	IsMember(ctx context.Context, chatID, userID int64) (bool, error)
	GetDialogBetweenUsers(ctx context.Context, user1ID, user2ID int64) (*domain.Chat, error)
}

type UserRepositoryInterface interface {
	Create(ctx context.Context, user *domainUser.User) (*domainUser.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domainUser.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domainUser.User, error)
	GetUserByID(ctx context.Context, id int64) (*domainUser.User, error)
}

type ChatService struct {
	chatRepo ChatRepositoryInterface
	userRepo UserRepositoryInterface
}

func NewChatService(chatRepo ChatRepositoryInterface, userRepo UserRepositoryInterface) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

func (s *ChatService) GetDialogName(ctx context.Context, chatID int64, userID int64) (string, error) {
	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("failed to get members: %w", err)
	}

	var friendUserID int64
	for _, member := range members {
		if member != userID {
			friendUserID = member
			break
		}
	}

	if friendUserID == 0 {
		return "", errors.New("no other user found in chat")
	}

	friendUser, err := s.userRepo.GetUserByID(ctx, friendUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get userID: %w", err)
	}

	return friendUser.Username, nil
}

func (s *ChatService) GetAllChats(ctx context.Context, id int64) ([]dto.ChatInformationDTO, error) {
	chats, err := s.chatRepo.GetAllChatsByUserID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}

	lastMessages, err := s.chatRepo.GetLastMessagesOfChats(ctx, id)
	if err != nil {
		lastMessages = []*domain.Message{}
	}

	lastMsgMap := make(map[int64]*domain.Message)
	for _, msg := range lastMessages {
		lastMsgMap[msg.ChatId] = msg
	}

	result := make([]dto.ChatInformationDTO, 0, len(chats))
	for _, chat := range chats {
		lastMsg, ok := lastMsgMap[chat.Id]
		messageDTO := dto.MessageDTO{}
		if ok && lastMsg != nil {
			messageDTO = dto.MessageDTO{
				SenderId: lastMsg.SenderId,
				Text: lastMsg.Content,
				CreatedAt: lastMsg.CreatedAt,
			}
		}
		displayTitle := chat.Title
		if chat.Type == domain.ChatTypeDialog {
			dialogName, err := s.GetDialogName(ctx, chat.Id, id)
			if err == nil && dialogName != "" {
				displayTitle = dialogName
			}
		}

		result = append(result, dto.ChatInformationDTO{
			ID: chat.Id,
			Title: displayTitle,
			ChatType: dto.ChatType(chat.Type),
			LastMessage: messageDTO,
		})
	}

	return result, nil
}

func (s *ChatService) CreateChat(ctx context.Context, chatDTO dto.ChatCreate, ownerID int64) (*dto.ChatInformationDTO, error) {
	if !slices.Contains(chatDTO.MembersID, ownerID) {
		chatDTO.MembersID = append(chatDTO.MembersID, ownerID)
	}

	if chatDTO.Type == dto.ChatTypeDialog && len(chatDTO.MembersID) != 2 {
		return nil, domain.ErrDialogMustHave2Users
	}

	if chatDTO.Type == dto.ChatTypeDialog && len(chatDTO.MembersID) == 2 {
		user1 := chatDTO.MembersID[0]
		user2 := chatDTO.MembersID[1]

		if user1 == user2 {
        	return nil, domain.ErrCantCreateDialogWithYourself
    	}

		existingDialog, err := s.chatRepo.GetDialogBetweenUsers(ctx, user1, user2)
		if err != nil {
			return nil, fmt.Errorf("failed to check dialog between users: %w", err)
		}

		if existingDialog != nil {
			return nil, domain.ErrDialogAlreadyExists
		}
	}

	title := chatDTO.Title
	if chatDTO.Type == dto.ChatTypeDialog {
		title = ""
	}

	chat := &domain.Chat{
		Type: domain.ChatType(chatDTO.Type),
		Title: title,
		Description: nil,
		OwnerId: ownerID,
		AvatarUrl: nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdChat, err := s.chatRepo.CreateChat(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	for _, member := range chatDTO.MembersID {
		err := s.chatRepo.AddMember(ctx, createdChat.Id, member)
		if err != nil {
			return nil, fmt.Errorf("failed to add member to chat: %w", err)
		}
	}

	displayTitle := createdChat.Title
	if createdChat.Type == domain.ChatTypeDialog {
		dialogName, err := s.GetDialogName(ctx, createdChat.Id, ownerID)
		if err == nil && dialogName != "" {
			displayTitle = dialogName
		}
	}

	return &dto.ChatInformationDTO{
		ID: createdChat.Id,
		ChatType: dto.ChatType(createdChat.Type),
		Title: displayTitle,
		LastMessage: dto.MessageDTO{},
	}, nil
}

func (s *ChatService) GetChatByID(ctx context.Context, chatID, userID int64) (*dto.ChatInformationDTO, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}
	

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to get chat by id: %w", err)
	}

	lastMsg, err := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNoMessage) {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	messageDTO := dto.MessageDTO{
		SenderId: lastMsg.SenderId,
		Text: lastMsg.Content,
		CreatedAt: lastMsg.CreatedAt,
	}

	displayTitle := chat.Title
	if chat.Type == domain.ChatTypeDialog {
		dialogName, err := s.GetDialogName(ctx, chat.Id, userID)
		if err == nil && dialogName != "" {
			displayTitle = dialogName
		}
	}

	return &dto.ChatInformationDTO{
		ID: chat.Id,
		ChatType: dto.ChatType(chat.Type),
		Title: displayTitle,
		LastMessage: messageDTO,
	}, nil

}