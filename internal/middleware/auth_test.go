package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/google/uuid"
)

type stubSessionService struct {
	getUserIDFunc func(sessionID string) (uuid.UUID, error)
}

func (s *stubSessionService) CreateSession(userID uuid.UUID) (*dtoSession.SessionDTO, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSessionService) GetUserID(sessionID string) (uuid.UUID, error) {
	if s.getUserIDFunc != nil {
		return s.getUserIDFunc(sessionID)
	}
	return uuid.New(), nil
}

func (s *stubSessionService) DeleteSession(sessionID string) error {
	return nil
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	mw := AuthMiddleware(&stubSessionService{})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	if called {
		t.Fatalf("expected next handler not to be called")
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	mw := AuthMiddleware(&stubSessionService{
		getUserIDFunc: func(sessionID string) (uuid.UUID, error) {
			return uuid.Nil, errors.New("invalid session")
		},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad"})

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	if called {
		t.Fatalf("expected next handler not to be called")
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	userID := uuid.New()
	mw := AuthMiddleware(&stubSessionService{
		getUserIDFunc: func(sessionID string) (uuid.UUID, error) {
			if sessionID != "good" {
				return uuid.Nil, errors.New("unexpected session id")
			}
			return userID, nil
		},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		ctxUserID, ok := r.Context().Value(UserID).(uuid.UUID)
		if !ok || ctxUserID != userID {
			t.Fatalf("expected userID %s in context, got %v", userID, ctxUserID)
		}

		ctxSessionID, ok := r.Context().Value(SessionID).(string)
		if !ok || ctxSessionID != "good" {
			t.Fatalf("expected sessionID 'good' in context, got %v", ctxSessionID)
		}
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "good"})

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if !called {
		t.Fatalf("expected next handler to be called")
	}
}
