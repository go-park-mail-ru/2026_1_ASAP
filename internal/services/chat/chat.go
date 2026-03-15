package chat

import (
	"slices"
	"errors"
	"time"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	dtoUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/user"
	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/chat"
	modelsUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	"github.com/google/uuid"
)

var (
	ErrAccessDenied = errors.New("You don't have access to this chat")
)

type ChatServiceInterface interface {
	GetAllChats(userID uuid.UUID) ([]dto.ChatInformationDTO, error)
	CreateChat(chatDTO dto.ChatCreate, ownerID uuid.UUID) (*dto.ChatInformationDTO, error)
	GetChatByID(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error)
}

type ChatService struct {
	chatRepository ChatRepository
	userRepository UserRepository
}

func NewChatService(chatRepository ChatRepository, userRepository UserRepository) *ChatService {
	return &ChatService{
		chatRepository: chatRepository,
		userRepository: userRepository,
	}
}

func (s *ChatService) GetAllChats(userID uuid.UUID) ([]dto.ChatInformationDTO, error) {
	chats, err := s.chatRepository.GetAllChatsByUserID(userID)
	if err != nil {
		return nil, err
	}

	lastMessages, err := s.chatRepository.GetLastMessagesOfChats(userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ChatInformationDTO, 0, len(chats))
	for i := range chats {
		userLogin, err := s.userRepository.GetUserByID(lastMessages[i].UserID)
		if err != nil {
			return nil, err
		}
		message := dto.MessageDTO{
			Sender:    dtoUser.UserDTO{Login: userLogin.Login},
			Text:      lastMessages[i].Text,
			CreatedAt: lastMessages[i].CreatedAt,
		}
		result = append(result, dto.ChatInformationDTO{
			ID:          chats[i].ID,
			Title:       chats[i].Title,
			LastMessage: message,
			ChatType:    dto.ChatType(chats[i].Type),
		})
	}

	return result, nil
}

func (s *ChatService) CreateChat(chatDTO dto.ChatCreate, ownerID uuid.UUID) (*dto.ChatInformationDTO, error) {
	owner := slices.Contains(chatDTO.MembersID, ownerID)
	if !owner {
		chatDTO.MembersID = append(chatDTO.MembersID, ownerID)
	}

	chat := &models.Chat{
		ID:        uuid.New(),
		Type:      models.ChatType(chatDTO.Type),
		Title:     chatDTO.Title,
		MembersID: chatDTO.MembersID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.chatRepository.CreateChat(chat)
	if err != nil {
		return nil, err
	}

	return &dto.ChatInformationDTO{
		ID:          chat.ID,
		Title:       chat.Title,
		LastMessage: dto.MessageDTO{},
		ChatType:    dto.ChatType(chat.Type),
	}, nil
}

func (s *ChatService) GetChatByID(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error) {
	chats, err := s.chatRepository.GetAllChatsByUserID(userID)
	if err != nil {
		return nil, err
	}

	for _, chat := range chats {
		if chat.ID == chatID {
			lasMessage, err := s.chatRepository.GetLastMessageOfChat(chat.ID)
			if err != nil {
				return nil, err
			}
			userLogin, err := s.userRepository.GetUserByID(lasMessage.UserID)
			if err != nil {
				return nil, err
			}
			message := &dto.MessageDTO{
				Sender:    dtoUser.UserDTO{Login: userLogin.Login},
				Text:      lasMessage.Text,
				CreatedAt: lasMessage.CreatedAt,
			}

			return &dto.ChatInformationDTO{
				ID:          chat.ID,
				Title:       chat.Title,
				LastMessage: *message,
				ChatType:    dto.ChatType(chat.Type),
			}, nil
		}
	}

	return nil, ErrAccessDenied
}
