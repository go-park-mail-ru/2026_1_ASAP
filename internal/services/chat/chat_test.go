package chat

import (
	"errors"
	"testing"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	chatRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/google/uuid"
)

func newTestChatService(t *testing.T) *ChatService {
	t.Helper()

	chatRepo := chatRepository.NewMockRepository()
	userRepo := userRepository.NewMockUserRepository()

	return NewChatService(chatRepo, userRepo)
}

func TestChatServiceGetAllChats_Success(t *testing.T) {
	service := newTestChatService(t)

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	chats, err := service.GetAllChats(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(chats) == 0 {
		t.Fatalf("expected at least one chat, got %d", len(chats))
	}
}

func TestChatServiceCreateChat_Success(t *testing.T) {
	service := newTestChatService(t)

	userID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	req := dto.ChatCreate{
		Title:     "New chat",
		Type:      dto.ChatTypeDialog,
		MembersID: []uuid.UUID{userID1, userID2},
	}

	chatInfo, err := service.CreateChat(req, userID1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if chatInfo == nil || chatInfo.ID == uuid.Nil {
		t.Fatalf("expected chat info with valid ID, got %#v", chatInfo)
	}

	if chatInfo.Title != req.Title {
		t.Fatalf("expected title %q, got %q", req.Title, chatInfo.Title)
	}
}

func TestChatServiceGetChatByID_Success(t *testing.T) {
	service := newTestChatService(t)

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	chatID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	chatInfo, err := service.GetChatByID(chatID, userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if chatInfo == nil || chatInfo.ID != chatID {
		t.Fatalf("expected chat with ID %s, got %#v", chatID, chatInfo)
	}
}

func TestChatServiceGetChatByID_AccessDenied(t *testing.T) {
	service := newTestChatService(t)

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	chatID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	chatInfo, err := service.GetChatByID(chatID, userID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if chatInfo != nil {
		t.Fatalf("expected nil chat info, got %#v", chatInfo)
	}

	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}
