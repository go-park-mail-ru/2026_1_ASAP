package chat

import (
	"errors"
	"sync"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/chat"
	"github.com/google/uuid"
)

var (
	ErrChatNotFound = errors.New("Chat not found")
	ErrChatAlreadyExists = errors.New("Chat already exists")
)

type ChatRepositoryInterface interface{
	GetAllChatsByUserID(userID uuid.UUID) ([]*models.Chat, error)
	GetChatByID(chatID uuid.UUID) (*models.Chat, error)
	CreateChat(chat *models.Chat) error
	GetLastMessagesOfChats(userID uuid.UUID) ([]*models.Message, error)
	GetLastMessageOfChat(chatID uuid.UUID) (*models.Message, error)
}

type ChatRepository struct {
	chats map[uuid.UUID]*models.Chat
	userInfo map[uuid.UUID][]uuid.UUID
	messages map[uuid.UUID][]*models.Message
	mu sync.RWMutex
}

func NewChatRepository() *ChatRepository{
	return &ChatRepository{
		chats: make(map[uuid.UUID]*models.Chat),
		userInfo: make(map[uuid.UUID][]uuid.UUID),
		messages: make(map[uuid.UUID][]*models.Message),
	}
}

func (c *ChatRepository) GetAllChatsByUserID(userID uuid.UUID) ([]*models.Chat, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userInfo, ok := c.userInfo[userID]
	if !ok {
		return make([]*models.Chat, 0), nil
	}

	result := make([]*models.Chat, 0, len(userInfo))

	for id := range userInfo {
		result = append(result, c.chats[userInfo[id]])
	}

	return result, nil
}

func (c *ChatRepository) GetChatByID(chatID uuid.UUID) (*models.Chat, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chat, ok := c.chats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	} 

	return chat, nil
}

func (c *ChatRepository) CreateChat(chat *models.Chat) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.chats[chat.ID]; exists {
		return ErrChatAlreadyExists 
	}

	c.chats[chat.ID] = chat
	for _, memberID := range chat.MembersID {
		c.userInfo[memberID] = append(c.userInfo[memberID], chat.ID)
	}

	return nil
}

func (c *ChatRepository) GetLastMessagesOfChats(userID uuid.UUID) ([]*models.Message, error) {
	chats, err := c.GetAllChatsByUserID(userID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	lastMessages := make([]*models.Message, 0, len(chats))
	for _, chat := range chats {
		chatMessages := c.messages[chat.ID]
		if len(chatMessages) == 0 {
			lastMessages = append(lastMessages, &models.Message{})
			continue
		}
		lastMessages = append(lastMessages, chatMessages[len(chatMessages)-1])
	}

	return lastMessages, nil
}

func (c *ChatRepository) GetLastMessageOfChat(chatID uuid.UUID) (*models.Message, error) {
	chat, err := c.GetChatByID(chatID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	chatMessages := c.messages[chat.ID]
	if len(chatMessages) == 0 {
		return &models.Message{}, nil
	}

	return chatMessages[len(chatMessages)-1], nil
}

