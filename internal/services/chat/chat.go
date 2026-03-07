package chat

import (
	"time"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/chat"
	chatRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat"
	"github.com/google/uuid"
)

type ChatServiceInterface interface {
	GetAllChats(userID uuid.UUID) ([]dto.ChatInformationDTO, error)
	CreateChat(chatDTO dto.ChatCreate) (*dto.ChatInformationDTO, error)
}

type ChatService struct {
	chatRepository chatRepository.ChatRepositoryInterface
}

func NewChatService(chatRepository chatRepository.ChatRepositoryInterface) *ChatService {
	return &ChatService{
		chatRepository: chatRepository,
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
		message := dto.MessageDTO{
			Sender:    lastMessages[i].UserID,
			Text:      lastMessages[i].Text,
			CreatedAt: lastMessages[i].CreatedAt,
		}
		result = append(result, dto.ChatInformationDTO{
			ID:          chats[i].ID,
			Title:       chats[i].Title,
			LastMessage: message,
		})
	}

	return result, nil
}

func (s *ChatService) CreateChat(chatDTO dto.ChatCreate) (*dto.ChatInformationDTO, error) {
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
	}, nil
}
