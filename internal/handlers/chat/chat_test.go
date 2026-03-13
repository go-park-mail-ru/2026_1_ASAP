package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/google/uuid"
)

type stubChatService struct {
	getAllFunc  func(userID uuid.UUID) ([]dto.ChatInformationDTO, error)
	createFunc  func(c dto.ChatCreate, ownerID uuid.UUID) (*dto.ChatInformationDTO, error)
	getByIDFunc func(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error)
}

func (s *stubChatService) GetAllChats(userID uuid.UUID) ([]dto.ChatInformationDTO, error) {
	if s.getAllFunc != nil {
		return s.getAllFunc(userID)
	}
	return []dto.ChatInformationDTO{}, nil
}

func (s *stubChatService) CreateChat(c dto.ChatCreate, ownerID uuid.UUID) (*dto.ChatInformationDTO, error) {
	if s.createFunc != nil {
		return s.createFunc(c, ownerID)
	}
	return &dto.ChatInformationDTO{ID: uuid.New(), Title: c.Title}, nil
}

func (s *stubChatService) GetChatByID(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(chatID, userID)
	}
	return &dto.ChatInformationDTO{ID: chatID, Title: "title"}, nil
}

func TestChatsHandlerGetChats_Unauthorized(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	rr := httptest.NewRecorder()

	h.GetChats(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestChatsHandlerGetChats_Success(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	ctx := context.WithValue(req.Context(), middleware.UserID, uuid.New())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.GetChats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestChatsHandlerChatCreate_InvalidJSON(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewReader([]byte("{invalid")))
	ctx := context.WithValue(req.Context(), middleware.UserID, uuid.New())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ChatCreate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestChatsHandlerChatCreate_Success(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	userID := uuid.New()
	body, _ := json.Marshal(dto.ChatCreate{
		Title:     "title",
		Type:      dto.ChatTypeGroup,
		MembersID: []uuid.UUID{userID},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ChatCreate(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
}

func TestChatsHandlerGetChatByID_Unauthorized(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/some", nil)
	rr := httptest.NewRecorder()

	h.GetChatByID(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestChatsHandlerGetChatByID_InvalidID(t *testing.T) {
	h := NewChatHandler(&stubChatService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/invalid", nil)
	ctx := context.WithValue(req.Context(), middleware.UserID, uuid.New())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// chi.URLParam reads from context set by router; here we simulate invalid UUID
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"id"},
			Values: []string{"invalid"},
		},
	}))

	h.GetChatByID(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestChatsHandlerGetChatByID_Forbidden(t *testing.T) {
	h := NewChatHandler(&stubChatService{
		getByIDFunc: func(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error) {
			return nil, errors.New("access denied")
		},
	})

	chatID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chatID.String(), nil)
	ctx := context.WithValue(req.Context(), middleware.UserID, uuid.New())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// simulate router param
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"id"},
			Values: []string{chatID.String()},
		},
	}))

	h.GetChatByID(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
	}
}
