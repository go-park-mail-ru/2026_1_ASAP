package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	modelsSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type stubAuthService struct {
	loginFunc    func(*dtoAuth.RequestLogin) (*modelsSession.SessionData, error)
	registerFunc func(*dtoAuth.RequestRegistrate) (*modelsSession.SessionData, []validation.ValidationError)
	logoutFunc   func(*dtoAuth.RequestLogout) error
}

func (s *stubAuthService) Register(r *dtoAuth.RequestRegistrate) (*modelsSession.SessionData, []validation.ValidationError) {
	if s.registerFunc != nil {
		return s.registerFunc(r)
	}
	return &modelsSession.SessionData{SessionID: "sid", Expire: time.Now().Add(time.Hour)}, nil
}

func (s *stubAuthService) Login(r *dtoAuth.RequestLogin) (*modelsSession.SessionData, error) {
	if s.loginFunc != nil {
		return s.loginFunc(r)
	}
	return &modelsSession.SessionData{SessionID: "sid", Expire: time.Now().Add(time.Hour)}, nil
}

func (s *stubAuthService) Logout(r *dtoAuth.RequestLogout) error {
	if s.logoutFunc != nil {
		return s.logoutFunc(r)
	}
	return nil
}

func TestAuthHandlerLogin_Success(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	body, _ := json.Marshal(dtoAuth.RequestLogin{
		Login:    "user",
		Password: "Passw0rd&",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestAuthHandlerLogin_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("{invalid")))
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestAuthHandlerRegister_Success(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	body, _ := json.Marshal(dtoAuth.RequestRegistrate{
		Login:    "user",
		Email:    "user@example.com",
		Password: "Passw0rd&",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.Register(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestAuthHandlerRegister_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("{invalid")))
	rr := httptest.NewRecorder()

	handler.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestAuthHandlerLogout_Unauthorized(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestAuthHandlerLogout_Success(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	ctx := context.WithValue(req.Context(), middleware.SessionID, "sid")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}
