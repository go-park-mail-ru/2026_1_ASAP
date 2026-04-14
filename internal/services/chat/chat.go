package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	domainProfile "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
)

//go:generate sh -c "mockgen -destination=mock/chat_mock.go -package=mock . ChatRepositoryInterface,UserRepositoryInterface,MediaRepositoryInterface && sed -i '' 's/chat \\*chat\\.Chat/chat \\*domain\\.Chat/g' mock/chat_mock.go && sed -i '' 's/user \\*user\\.User/u \\*domainUser\\.User/g' mock/chat_mock.go"
type ChatRepositoryInterface interface {
	GetAllChatsByUserID(ctx context.Context, id int64) ([]*domain.Chat, error)
	GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error)
	CreateChat(ctx context.Context, chat *domain.Chat) (*domain.Chat, error)
	GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error)
	GetLastMessagesOfChats(ctx context.Context, id int64) ([]*domain.Message, error)
	GetChatMembers(ctx context.Context, chatID int64) ([]int64, error)
	AddMember(ctx context.Context, chatID, userID int64, role string) error
	IsMember(ctx context.Context, chatID, userID int64) (bool, error)
	GetDialogBetweenUsers(ctx context.Context, user1ID, user2ID int64) (*domain.Chat, error)
	DeleteChat(ctx context.Context, chatID int64) error
	GetMemberRole(ctx context.Context, userID, chatID int64) (string, error)
	UploadAvatarUrl(ctx context.Context, chatID int64, avatarURL string) (*domain.Chat, error)
	UpdateTitle(ctx context.Context, chatID int64, title string) (*domain.Chat, error)
	DeleteMember(ctx context.Context, chatID, userID int64) error
}

type UserRepositoryInterface interface {
	Create(ctx context.Context, user *domainUser.User) (*domainUser.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domainUser.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domainUser.User, error)
	GetUserByID(ctx context.Context, id int64) (*domainUser.User, error)
}

type MediaRepositoryInterface interface {
	UploadChatAvatar(ctx context.Context, chatID int64, input *media.FileInput) (string, error)
}

type ChatService struct {
	chatRepo  ChatRepositoryInterface
	userRepo  UserRepositoryInterface
	mediaRepo MediaRepositoryInterface
}

func NewChatService(chatRepo ChatRepositoryInterface, userRepo UserRepositoryInterface, mediaRepo MediaRepositoryInterface) *ChatService {
	return &ChatService{
		chatRepo:  chatRepo,
		userRepo:  userRepo,
		mediaRepo: mediaRepo,
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

	return friendUser.Username(), nil
}

func (s *ChatService) GetDialogAvatar(ctx context.Context, chatID int64, userID int64) (*string, error) {
	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	var friendUserID int64
	for _, member := range members {
		if member != userID {
			friendUserID = member
			break
		}
	}

	if friendUserID == 0 {
		return nil, errors.New("no other user found in chat")
	}

	friendUser, err := s.userRepo.GetUserByID(ctx, friendUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get userID: %w", err)
	}

	return friendUser.AvatarUrl, nil
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
				SenderId:  lastMsg.SenderId,
				Text:      lastMsg.Content,
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
		displayAvatar := chat.AvatarUrl
		if chat.Type == domain.ChatTypeDialog {
			dialogAvatar, err := s.GetDialogAvatar(ctx, chat.Id, id)
			if err == nil && dialogAvatar != nil {
				displayAvatar = dialogAvatar
			}
		}

		result = append(result, dto.ChatInformationDTO{
			ID:          chat.Id,
			Title:       displayTitle,
			ChatType:    dto.ChatType(chat.Type),
			LastMessage: messageDTO,
			Avatar:      displayAvatar,
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
		Type:        domain.ChatType(chatDTO.Type),
		Title:       title,
		Description: nil,
		OwnerId:     ownerID,
		AvatarUrl:   nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdChat, err := s.chatRepo.CreateChat(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	for _, member := range chatDTO.MembersID {
		_, err := s.userRepo.GetUserByID(ctx, member)
		if err != nil {
			if errors.Is(err, domainUser.ErrNotFound) {
				return nil, domainUser.ErrNotFound
			}
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		role := "member"
		if member == ownerID {
			role = "owner"
		}
		err = s.chatRepo.AddMember(ctx, createdChat.Id, member, role)
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

	displayAvatar := chat.AvatarUrl
	if createdChat.Type == domain.ChatTypeDialog {
		dialogAvatar, err := s.GetDialogAvatar(ctx, createdChat.Id, ownerID)
		if err == nil && dialogAvatar != nil {
			displayAvatar = dialogAvatar
		}
	}

	return &dto.ChatInformationDTO{
		ID:          createdChat.Id,
		ChatType:    dto.ChatType(createdChat.Type),
		Title:       displayTitle,
		LastMessage: dto.MessageDTO{},
		Avatar:      displayAvatar,
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
		SenderId:  lastMsg.SenderId,
		Text:      lastMsg.Content,
		CreatedAt: lastMsg.CreatedAt,
	}

	displayTitle := chat.Title
	if chat.Type == domain.ChatTypeDialog {
		dialogName, err := s.GetDialogName(ctx, chat.Id, userID)
		if err == nil && dialogName != "" {
			displayTitle = dialogName
		}
	}

	displayAvatar := chat.AvatarUrl
	if chat.Type == domain.ChatTypeDialog {
		dialogAvatar, err := s.GetDialogAvatar(ctx, chat.Id, userID)
		if err == nil && dialogAvatar != nil {
			displayAvatar = dialogAvatar
		}
	}

	return &dto.ChatInformationDTO{
		ID:          chat.Id,
		ChatType:    dto.ChatType(chat.Type),
		Title:       displayTitle,
		LastMessage: messageDTO,
		Avatar:      displayAvatar,
	}, nil
}

func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID int64) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return domain.ErrChatNotFound
		}
	}

	if chat.Type == domain.ChatTypeDialog {
		err = s.chatRepo.DeleteChat(ctx, chatID)
		if err != nil {
			return fmt.Errorf("failed to delete chat: %w", err)
		}

		return nil
	}

	userRole, err := s.chatRepo.GetMemberRole(ctx, userID, chatID)
	if err != nil {
		return fmt.Errorf("failed to check user role: %w", err)
	}
	if userRole == "" {
		return domain.ErrNotMember
	}
	if userRole != "owner" {
		return domain.ErrOnlyOwnerCanDeleteChat
	}

	err = s.chatRepo.DeleteChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	return nil
}

func (s *ChatService) IsMember(ctx context.Context, userID, chatID int64) (bool, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	return isMember, nil
}

func (s *ChatService) UpdateChatAvatar(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateAvatar) (*dto.ChatInformationDTO, error) {
	if request == nil {
		return nil, errors.New("update profile avatar nil request")
	}
	err := checkAvatar(request.File)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrFileTooLarge):
			return nil, domainProfile.ErrAvatarTooLarge
		case errors.Is(err, media.ErrInvalidFileType):
			return nil, domainProfile.ErrInvalidAvatarType
		case errors.Is(err, media.ErrEmptyFile):
			return nil, domainProfile.ErrEmptyAvatar
		}

		return nil, fmt.Errorf("invalid avatar: %w", err)
	}
	
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

	if chat.Type == domain.ChatTypeDialog {
		return nil, domain.ErrDialogCannotHaveCustomAvatar
	}

	avatarURL, err := s.mediaRepo.UploadChatAvatar(ctx, chatID, request.File)
	if err != nil {
		return nil, fmt.Errorf("failed upload avatar: %w", err)
	}

	result, err := s.chatRepo.UploadAvatarUrl(ctx, chatID, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("failed upload avatar url: %w", err)
	}

	lastMsg, err := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNoMessage) {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	messageDTO := dto.MessageDTO{
		SenderId:  lastMsg.SenderId,
		Text:      lastMsg.Content,
		CreatedAt: lastMsg.CreatedAt,
	}

	return &dto.ChatInformationDTO{
		ID:          result.Id,
		ChatType:    dto.ChatType(result.Type),
		Title:       result.Title,
		LastMessage: messageDTO,
		Avatar:      result.AvatarUrl,
	}, nil
}

func (s *ChatService) UpdateChatTitle(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateTitle) (*dto.ChatInformationDTO, error) {
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

	if chat.Type == domain.ChatTypeDialog {
		return nil, domain.ErrDialogCannotHaveCustomTitle
	}

	result, err := s.chatRepo.UpdateTitle(ctx, chatID, request.Title)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to update title: %w", err)
	}

	lastMsg, err := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNoMessage) {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	messageDTO := dto.MessageDTO{
		SenderId:  lastMsg.SenderId,
		Text:      lastMsg.Content,
		CreatedAt: lastMsg.CreatedAt,
	}

	return &dto.ChatInformationDTO{
		ID:          result.Id,
		ChatType:    dto.ChatType(result.Type),
		Title:       result.Title,
		LastMessage: messageDTO,
		Avatar:      result.AvatarUrl,
	}, nil
}

func (s *ChatService) AddMembersToChat(ctx context.Context, userID, chatID int64, request *dto.RequestAddMember) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return domain.ErrChatNotFound
		}

		return fmt.Errorf("failed to get chat by id: %w", err)
	}

	if chat.Type == domain.ChatTypeDialog {
		return domain.ErrCantAddMemberToDialog
	}
	if chat.OwnerId != userID {
		return domain.ErrOnlyOwnerCanAddPeople
	}

	for _, member := range request.MembersId {
		isMember, err := s.chatRepo.IsMember(ctx, chatID, member)
		if err != nil {
			return fmt.Errorf("failed to check user is member of chat: %w", err)
		}
		if isMember {
			return domain.ErrMemberAlreadyInChat
		}

		role := "member"
		err = s.chatRepo.AddMember(ctx, chatID, member, role)
		if err != nil {
			return fmt.Errorf("failed to add member to chat: %w", err)
		}
	}

	return nil
}

func (s *ChatService) DeleteMemberFromChat(ctx context.Context, userID, chatID int64, request *dto.RequestDeleteMember) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return domain.ErrChatNotFound
		}

		return fmt.Errorf("failed to get chat by id: %w", err)
	}

	if chat.Type == domain.ChatTypeDialog {
		return domain.ErrCantDeleteMemberFromDialog
	}
	if chat.OwnerId != userID {
		return domain.ErrOnlyOwnerCanDeletePeople
	}

	if request.MemberId == chat.OwnerId {
		return domain.ErrCantDeleteOwnerOfChat
	}
	memberExists, err := s.chatRepo.IsMember(ctx, chatID, request.MemberId)
	if err != nil {
		return fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !memberExists {
		return domain.ErrUserNotMember
	}

	err = s.chatRepo.DeleteMember(ctx, chatID, request.MemberId)
	if err != nil {
		if errors.Is(err, domain.ErrMemberNotFound) {
			return domain.ErrMemberNotFound
		}
		return err
	}

	return nil
}

func (s *ChatService) GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error) {
	return s.chatRepo.GetChatMembers(ctx, chatID)
}

func (s *ChatService) GetAllChatMembers(ctx context.Context, userID, chatID int64) (*dto.ResponseGetChatMembers, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all chat members: %w", err)
	}

	return &dto.ResponseGetChatMembers{
		MembersId: members,
	}, nil
}

func (s *ChatService) QuitChat(ctx context.Context, userID, chatID int64) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user is member of chat: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return domain.ErrChatNotFound
		}

		return fmt.Errorf("failed to get chat by id: %w", err)
	}

	if chat.Type == domain.ChatTypeDialog {
		return domain.ErrCantQuitDialog
	}
	if chat.OwnerId == userID {
		return domain.ErrOwnerCantQuitGroup
	}

	err = s.chatRepo.DeleteMember(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrMemberNotFound) {
			return domain.ErrMemberNotFound
		}
		return err
	}

	return nil
}

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func checkAvatar(input *media.FileInput) error {
	if input == nil || input.Body == nil {
		return media.ErrEmptyFile
	}
	if input.Size <= 0 {
		return media.ErrEmptyFile
	}
	if input.Size > media.MaxAvatarBytes {
		return media.ErrFileTooLarge
	}
	if !allowedAvatarTypes[input.ContentType] {
		return media.ErrInvalidFileType
	}
	return nil
}
