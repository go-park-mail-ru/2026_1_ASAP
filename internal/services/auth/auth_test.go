package auth

import (
	"testing"
	"time"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
)

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()

	userRepo := userRepository.NewMockUserRepository()
	sessionRepo := sessions.NewSessionRepository()
	sessionService := session.NewSessionService(sessionRepo, time.Hour)

	return NewAuthService(userRepo, sessionService)
}

func TestAuthServiceRegister_Success(t *testing.T) {
	authService := newTestAuthService(t)

	req := &dtoAuth.RequestRegistrate{
		Login:    "newuser",
		Email:    "newuser@example.com",
		Password: "Passw0rd&",
	}

	sessionData, errs := authService.Register(req)

	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}

	if sessionData == nil || sessionData.SessionID == "" {
		t.Fatalf("expected non nil session data with id, got %#v", sessionData)
	}
}

func TestAuthServiceRegister_EmailAlreadyRegistered(t *testing.T) {
	authService := newTestAuthService(t)

	// email already exists in mock repository
	req := &dtoAuth.RequestRegistrate{
		Login:    "newlogin",
		Email:    "alice@example.com",
		Password: "Passw0rd&",
	}

	sessionData, errs := authService.Register(req)

	if sessionData != nil {
		t.Fatalf("expected nil session data, got %#v", sessionData)
	}

	if len(errs) != 1 || errs[0].Code != "EMAIL_ALREADY_REGISTERED" {
		t.Fatalf("expected EMAIL_ALREADY_REGISTERED error, got %#v", errs)
	}
}

func TestAuthServiceRegister_LoginAlreadyRegistered(t *testing.T) {
	authService := newTestAuthService(t)

	// login already exists in mock repository
	req := &dtoAuth.RequestRegistrate{
		Login:    "alice",
		Email:    "newalice@example.com",
		Password: "Passw0rd&",
	}

	sessionData, errs := authService.Register(req)

	if sessionData != nil {
		t.Fatalf("expected nil session data, got %#v", sessionData)
	}

	if len(errs) != 1 || errs[0].Code != "LOGIN_ALREADY_REGISTERED" {
		t.Fatalf("expected LOGIN_ALREADY_REGISTERED error, got %#v", errs)
	}
}

func TestAuthServiceLogin_Success(t *testing.T) {
	authService := newTestAuthService(t)

	req := &dtoAuth.RequestLogin{
		Login:    "alice",
		Password: "passWo1r&",
	}

	sessionData, err := authService.Login(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sessionData == nil || sessionData.SessionID == "" {
		t.Fatalf("expected non nil session data with id, got %#v", sessionData)
	}
}

func TestAuthServiceLogin_InvalidLogin(t *testing.T) {
	authService := newTestAuthService(t)

	req := &dtoAuth.RequestLogin{
		Login:    "unknown",
		Password: "passWo1r&",
	}

	sessionData, err := authService.Login(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if sessionData != nil {
		t.Fatalf("expected nil session data, got %#v", sessionData)
	}
}

func TestAuthServiceLogin_InvalidPassword(t *testing.T) {
	authService := newTestAuthService(t)

	req := &dtoAuth.RequestLogin{
		Login:    "alice",
		Password: "wrongPassword1!",
	}

	sessionData, err := authService.Login(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if sessionData != nil {
		t.Fatalf("expected nil session data, got %#v", sessionData)
	}
}

func TestAuthServiceLogout_Success(t *testing.T) {
	authService := newTestAuthService(t)

	req := &dtoAuth.RequestLogout{
		SessionID: "some-session-id",
	}

	if err := authService.Logout(req); err != nil {
		t.Fatalf("expected no error on logout, got %v", err)
	}
}
