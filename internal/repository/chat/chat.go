package chat

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
)

var (
	ErrChatNotFound      = errors.New("Chat not found")
	ErrChatAlreadyExists = errors.New("Chat already exists")
)

type ChatRepository struct {
	chats    map[uuid.UUID]*domain.Chat
	userInfo map[uuid.UUID][]uuid.UUID
	messages map[uuid.UUID][]*domain.Message
	mu       sync.RWMutex
}

func NewChatRepository() *ChatRepository {
	return &ChatRepository{
		chats:    make(map[uuid.UUID]*domain.Chat),
		userInfo: make(map[uuid.UUID][]uuid.UUID),
		messages: make(map[uuid.UUID][]*domain.Message),
	}
}

func (c *ChatRepository) GetAllChatsByUserID(userID uuid.UUID) ([]*domain.Chat, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userInfo, ok := c.userInfo[userID]
	if !ok {
		return make([]*domain.Chat, 0), nil
	}

	result := make([]*domain.Chat, 0, len(userInfo))

	for id := range userInfo {
		result = append(result, c.chats[userInfo[id]])
	}

	return result, nil
}

func (c *ChatRepository) GetChatByID(chatID uuid.UUID) (*domain.Chat, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chat, ok := c.chats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}

	return chat, nil
}

func (c *ChatRepository) CreateChat(chat *domain.Chat) error {
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

func (c *ChatRepository) GetLastMessagesOfChats(userID uuid.UUID) ([]*domain.Message, error) {
	chats, err := c.GetAllChatsByUserID(userID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	lastMessages := make([]*domain.Message, 0, len(chats))
	for _, chat := range chats {
		chatMessages := c.messages[chat.ID]
		if len(chatMessages) == 0 {
			lastMessages = append(lastMessages, &domain.Message{})
			continue
		}
		lastMessages = append(lastMessages, chatMessages[len(chatMessages)-1])
	}

	return lastMessages, nil
}

func (c *ChatRepository) GetLastMessageOfChat(chatID uuid.UUID) (*domain.Message, error) {
	chat, err := c.GetChatByID(chatID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	chatMessages := c.messages[chat.ID]
	if len(chatMessages) == 0 {
		return &domain.Message{}, nil
	}

	return chatMessages[len(chatMessages)-1], nil
}
