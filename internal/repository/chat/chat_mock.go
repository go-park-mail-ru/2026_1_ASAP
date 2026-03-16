package chat

import (
	"sync"
	"time"

	"github.com/google/uuid"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/chat"
)

func NewMockRepository() *ChatRepository {
	now := time.Now()

	// пользователи (UUID + логины)
	user1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	user3 := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	// чаты
	chat1ID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	chat2ID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	chat3ID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	chat1 := &models.Chat{
		ID:        chat1ID,
		Type:      models.ChatTypeDialog,
		Title:     "Dialog 1",
		MembersID: []uuid.UUID{user1, user2},
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	}

	chat2 := &models.Chat{
		ID:        chat2ID,
		Type:      models.ChatTypeGroup,
		Title:     "Backend Team",
		MembersID: []uuid.UUID{user1, user2, user3},
		CreatedAt: now.Add(-48 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}

	chat3 := &models.Chat{
		ID:        chat3ID,
		Type:      models.ChatTypeChannel,
		Title:     "Announcements",
		MembersID: []uuid.UUID{user1, user3},
		CreatedAt: now.Add(-72 * time.Hour),
		UpdatedAt: now.Add(-30 * time.Minute),
	}

	// сообщения
	msg1 := &models.Message{
		ID:        uuid.New(),
		ChatID:    chat1ID,
		UserID:    user1,
		Text:      "Hello!",
		CreatedAt: now.Add(-10 * time.Minute),
	}

	msg2 := &models.Message{
		ID:        uuid.New(),
		ChatID:    chat1ID,
		UserID:    user2,
		Text:      "Hi! How are you?",
		CreatedAt: now.Add(-8 * time.Minute),
	}

	msg3 := &models.Message{
		ID:        uuid.New(),
		ChatID:    chat2ID,
		UserID:    user3,
		Text:      "Don't forget the deploy today",
		CreatedAt: now.Add(-5 * time.Minute),
	}

	msg4 := &models.Message{
		ID:        uuid.New(),
		ChatID:    chat2ID,
		UserID:    user2,
		Text:      "All set for release",
		CreatedAt: now.Add(-3 * time.Minute),
	}

	msg5 := &models.Message{
		ID:        uuid.New(),
		ChatID:    chat3ID,
		UserID:    user1,
		Text:      "Welcome to the channel!",
		CreatedAt: now.Add(-25 * time.Minute),
	}

	return &ChatRepository{
		chats: map[uuid.UUID]*models.Chat{
			chat1ID: chat1,
			chat2ID: chat2,
			chat3ID: chat3,
		},

		userInfo: map[uuid.UUID][]uuid.UUID{
			user1: {chat1ID, chat2ID, chat3ID},
			user2: {chat1ID, chat2ID},
			user3: {chat2ID, chat3ID},
		},

		messages: map[uuid.UUID][]*models.Message{
			chat1ID: {msg1, msg2},
			chat2ID: {msg3, msg4},
			chat3ID: {msg5},
		},

		mu: sync.RWMutex{},
	}
}
