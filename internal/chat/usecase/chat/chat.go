package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=chat.go -destination=mock/chat_mock.go -package=mock
type ChatRepositoryInterface interface {
	GetAllChatsByUserID(ctx context.Context, id int64) ([]*domain.Chat, error)
	GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error)
	CreateChat(ctx context.Context, newChat *domain.Chat) (*domain.Chat, error)
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
	UpdateDescription(ctx context.Context, chatID int64, description string) (*domain.Chat, error)
}

type ProfileServiceInterface interface {
	GetUserByID(ctx context.Context, id int64) (*pdomain.Profile, error)
}

type MediaRepositoryInterface interface {
	UploadChatAvatar(ctx context.Context, chatID int64, input *media.FileInput) (string, error)
}

type ChatService struct {
	chatRepo  ChatRepositoryInterface
	userSvc   ProfileServiceInterface
	mediaRepo MediaRepositoryInterface
}

func NewChatService(chatRepo ChatRepositoryInterface, userSvc ProfileServiceInterface, mediaRepo MediaRepositoryInterface) *ChatService {
	return &ChatService{
		chatRepo:  chatRepo,
		userSvc:   userSvc,
		mediaRepo: mediaRepo,
	}
}

func (s *ChatService) getDialogName(ctx context.Context, chatID int64, userID int64) (string, error) {
	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("failed to get members: %w", err)
	}

	var friendID int64
	for _, m := range members {
		if m != userID {
			friendID = m
			break
		}
	}
	if friendID == 0 {
		return "", errors.New("no other user found in dialog")
	}

	user, err := s.userSvc.GetUserByID(ctx, friendID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.LastName != nil {
		return user.FirstName + " " + *user.LastName, nil
	}
	return user.FirstName, nil
}

func (s *ChatService) getDialogAvatar(ctx context.Context, chatID int64, userID int64) (*string, error) {
	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	var friendID int64
	for _, m := range members {
		if m != userID {
			friendID = m
			break
		}
	}
	if friendID == 0 {
		return nil, errors.New("no other user found in dialog")
	}

	user, err := s.userSvc.GetUserByID(ctx, friendID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user.Avatar, nil
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
		lastMsg := lastMsgMap[chat.Id]
		var messageDTO dto.MessageDTO
		if lastMsg != nil {
			messageDTO = dto.MessageDTO{
				SenderId:  lastMsg.SenderId,
				Text:      sanitize.Text(lastMsg.Content),
				CreatedAt: lastMsg.CreatedAt,
			}
		}

		displayTitle := chat.Title
		displayAvatar := chat.AvatarUrl

		if chat.Type == domain.ChatTypeDialog {
			if name, err := s.getDialogName(ctx, chat.Id, id); err == nil && name != "" {
				displayTitle = name
			}
			if avatar, err := s.getDialogAvatar(ctx, chat.Id, id); err == nil {
				displayAvatar = avatar
			}
		}

		result = append(result, dto.ChatInformationDTO{
			ID:          chat.Id,
			Title:       sanitize.Text(displayTitle),
			ChatType:    dto.ChatType(chat.Type),
			LastMessage: messageDTO,
			Avatar:      displayAvatar,
			OwnerID:     chat.OwnerId,
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

	if chatDTO.Type == dto.ChatTypeDialog {
		u1, u2 := chatDTO.MembersID[0], chatDTO.MembersID[1]
		if u1 == u2 {
			return nil, domain.ErrCantCreateDialogWithYourself
		}
		existing, err := s.chatRepo.GetDialogBetweenUsers(ctx, u1, u2)
		if err != nil && !errors.Is(err, domain.ErrChatNotFound) {
			return nil, fmt.Errorf("check dialog: %w", err)
		}
		if existing != nil {
			return nil, domain.ErrDialogAlreadyExists
		}
	}

	title := chatDTO.Title
	if chatDTO.Type == dto.ChatTypeDialog {
		title = ""
	}

	for _, memberID := range chatDTO.MembersID {
		if _, err := s.userSvc.GetUserByID(ctx, memberID); err != nil {
			return nil, fmt.Errorf("get user %d: %w", memberID, err)
		}
	}

	chat := &domain.Chat{
		Type:      domain.ChatType(chatDTO.Type),
		Title:     title,
		OwnerId:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	created, err := s.chatRepo.CreateChat(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("create chat: %w", err)
	}

	for _, memberID := range chatDTO.MembersID {
		role := "member"
		if memberID == ownerID {
			role = "owner"
		}
		if err := s.chatRepo.AddMember(ctx, created.Id, memberID, role); err != nil {
			return nil, fmt.Errorf("add member %d: %w", memberID, err)
		}
	}

	displayTitle := created.Title
	displayAvatar := created.AvatarUrl
	if created.Type == domain.ChatTypeDialog {
		if name, err := s.getDialogName(ctx, created.Id, ownerID); err == nil && name != "" {
			displayTitle = name
		}
		if avatar, err := s.getDialogAvatar(ctx, created.Id, ownerID); err == nil {
			displayAvatar = avatar
		}
	}

	return &dto.ChatInformationDTO{
		ID:          created.Id,
		ChatType:    dto.ChatType(created.Type),
		Title:       sanitize.Text(displayTitle),
		LastMessage: dto.MessageDTO{},
		Avatar:      displayAvatar,
		OwnerID:     created.OwnerId,
	}, nil
}

func (s *ChatService) GetChatByID(ctx context.Context, chatID, userID int64) (*dto.ChatInformationDTO, error) {
	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember && chat.Type != domain.ChatTypeChannel {
		return nil, domain.ErrNotMember
	}

	lastMsg, err := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	var messageDTO dto.MessageDTO
	if err == nil && lastMsg != nil {
		messageDTO = dto.MessageDTO{
			SenderId:  lastMsg.SenderId,
			Text:      sanitize.Text(lastMsg.Content),
			CreatedAt: lastMsg.CreatedAt,
		}
	}

	displayTitle := chat.Title
	displayAvatar := chat.AvatarUrl
	if chat.Type == domain.ChatTypeDialog {
		if name, err := s.getDialogName(ctx, chat.Id, userID); err == nil && name != "" {
			displayTitle = name
		}
		if avatar, err := s.getDialogAvatar(ctx, chat.Id, userID); err == nil {
			displayAvatar = avatar
		}
	}

	return &dto.ChatInformationDTO{
		ID:          chat.Id,
		ChatType:    dto.ChatType(chat.Type),
		Title:       sanitize.Text(displayTitle),
		LastMessage: messageDTO,
		Avatar:      displayAvatar,
		OwnerID:     chat.OwnerId,
	}, nil
}

func (s *ChatService) DeleteChat(ctx context.Context, userID, chatID int64) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return err
	}

	if chat.Type != domain.ChatTypeDialog {
		role, err := s.chatRepo.GetMemberRole(ctx, userID, chatID)
		if err != nil {
			return fmt.Errorf("get role: %w", err)
		}
		if role != "owner" {
			return domain.ErrOnlyOwnerCanDeleteChat
		}
	}

	return s.chatRepo.DeleteChat(ctx, chatID)
}

func (s *ChatService) UpdateChatAvatar(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateAvatar) (*dto.ChatInformationDTO, error) {
	if request == nil {
		return nil, errors.New("nil request")
	}
	if err := checkAvatar(request.File); err != nil {
		return nil, err
	}

	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if chat.Type == domain.ChatTypeChannel {
		if chat.OwnerId != userID {
			return nil, domain.ErrOnlyOwnerCanChangeAvatar
		}
	}

	if chat.Type == domain.ChatTypeDialog {
		return nil, domain.ErrDialogCannotHaveCustomAvatar
	}

	avatarURL, err := s.mediaRepo.UploadChatAvatar(ctx, chatID, request.File)
	if err != nil {
		return nil, fmt.Errorf("upload avatar: %w", err)
	}

	result, err := s.chatRepo.UploadAvatarUrl(ctx, chatID, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("save avatar url: %w", err)
	}

	lastMsg, _ := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	var messageDTO dto.MessageDTO
	if lastMsg != nil {
		messageDTO = dto.MessageDTO{
			SenderId:  lastMsg.SenderId,
			Text:      sanitize.Text(lastMsg.Content),
			CreatedAt: lastMsg.CreatedAt,
		}
	}

	return &dto.ChatInformationDTO{
		ID:          result.Id,
		ChatType:    dto.ChatType(result.Type),
		Title:       sanitize.Text(result.Title),
		LastMessage: messageDTO,
		Avatar:      result.AvatarUrl,
		OwnerID:     result.OwnerId,
	}, nil
}

func (s *ChatService) UpdateChatTitle(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateTitle) (*dto.ChatInformationDTO, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if chat.Type == domain.ChatTypeChannel {
		if chat.OwnerId != userID {
			return nil, domain.ErrOnlyOwnerCanChangeTitle
		}
	}

	if chat.Type == domain.ChatTypeDialog {
		return nil, domain.ErrDialogCannotHaveCustomTitle
	}

	result, err := s.chatRepo.UpdateTitle(ctx, chatID, request.Title)
	if err != nil {
		return nil, err
	}

	lastMsg, _ := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	var messageDTO dto.MessageDTO
	if lastMsg != nil {
		messageDTO = dto.MessageDTO{
			SenderId:  lastMsg.SenderId,
			Text:      sanitize.Text(lastMsg.Content),
			CreatedAt: lastMsg.CreatedAt,
		}
	}

	return &dto.ChatInformationDTO{
		ID:          result.Id,
		ChatType:    dto.ChatType(result.Type),
		Title:       sanitize.Text(result.Title),
		LastMessage: messageDTO,
		Avatar:      result.AvatarUrl,
		OwnerID:     result.OwnerId,
	}, nil
}

func (s *ChatService) UpdateChatDescription(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateDescription) (*dto.ChatInformationDTO, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if chat.Type == domain.ChatTypeChannel {
		if chat.OwnerId != userID {
			return nil, domain.ErrOnlyOwnerCanChangeDescription
		}
	}

	if chat.Type == domain.ChatTypeDialog {
		return nil, domain.ErrDialogCannotHaveCustomDescription
	}

	result, err := s.chatRepo.UpdateDescription(ctx, chatID, request.Description)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to update description: %w", err)
	}

	lastMsg, _ := s.chatRepo.GetLastMessageOfChat(ctx, chatID)
	var messageDTO dto.MessageDTO
	if lastMsg != nil {
		messageDTO = dto.MessageDTO{
			SenderId:  lastMsg.SenderId,
			Text:      sanitize.Text(lastMsg.Content),
			CreatedAt: lastMsg.CreatedAt,
		}
	}

	return &dto.ChatInformationDTO{
		ID:          result.Id,
		ChatType:    dto.ChatType(result.Type),
		Title:       sanitize.Text(result.Title),
		LastMessage: messageDTO,
		Avatar:      result.AvatarUrl,
		OwnerID:     result.OwnerId,
		Description: result.Description,
	}, nil
}

func (s *ChatService) AddMembersToChat(ctx context.Context, userID, chatID int64, request *dto.RequestAddMember) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return err
	}
	if chat.Type == domain.ChatTypeDialog {
		return domain.ErrCantAddMemberToDialog
	}
	if chat.OwnerId != userID {
		return domain.ErrOnlyOwnerCanAddPeople
	}

	for _, memberID := range request.MembersId {
		already, err := s.chatRepo.IsMember(ctx, chatID, memberID)
		if err != nil {
			return fmt.Errorf("check member %d: %w", memberID, err)
		}
		if already {
			return domain.ErrMemberAlreadyInChat
		}
		if err := s.chatRepo.AddMember(ctx, chatID, memberID, "member"); err != nil {
			return fmt.Errorf("add member %d: %w", memberID, err)
		}
	}
	return nil
}

func (s *ChatService) DeleteMemberFromChat(ctx context.Context, userID, chatID int64, request *dto.RequestDeleteMember) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return err
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

	exists, err := s.chatRepo.IsMember(ctx, chatID, request.MemberId)
	if err != nil {
		return fmt.Errorf("check target member: %w", err)
	}
	if !exists {
		return domain.ErrUserNotMember
	}

	return s.chatRepo.DeleteMember(ctx, chatID, request.MemberId)
}

func (s *ChatService) GetAllChatMembers(ctx context.Context, userID, chatID int64) (*dto.ResponseGetChatMembers, error) {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	members, err := s.chatRepo.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	return &dto.ResponseGetChatMembers{MembersId: members}, nil
}

func (s *ChatService) QuitChat(ctx context.Context, userID, chatID int64) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return domain.ErrNotMember
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return err
	}
	if chat.Type == domain.ChatTypeDialog {
		return domain.ErrCantQuitDialog
	}
	if chat.OwnerId == userID {
		return domain.ErrOwnerCantQuitGroup
	}

	return s.chatRepo.DeleteMember(ctx, chatID, userID)
}

func (s *ChatService) JoinChannel(ctx context.Context, userID, chatID int64) error {
	isMember, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if isMember {
		return nil
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return err
	}
	if chat.Type != domain.ChatTypeChannel {
		return domain.ErrCanJoinOnlyChannel
	}

	err = s.chatRepo.AddMember(ctx, chatID, userID, "member")
	if err != nil {
		return fmt.Errorf("failed to join chat: %w", err)
	}

	return nil
}

func (s *ChatService) GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error) {
	return s.chatRepo.GetChatMembers(ctx, chatID)
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
