package session

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	sessionModel "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
)

func TestSessionServiceCreateAndGetUserID_Success(t *testing.T) {
	repo := sessions.NewSessionRepository()
	service := NewSessionService(repo, time.Hour)

	userID := uuid.New()

	sessionData, err := service.CreateSession(userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sessionData.SessionID == "" {
		t.Fatalf("expected non-empty session id")
	}

	returnedUserID, err := service.GetUserID(sessionData.SessionID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if returnedUserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, returnedUserID)
	}
}

type expiredSessionRepo struct {
	SessionRepository
}

func (e *expiredSessionRepo) GetSession(_ string) (*sessionModel.Session, error) {
	return &sessionModel.Session{
		UserID: uuid.New(),
		Expire: time.Now().Add(-time.Minute),
	}, nil
}

func (e *expiredSessionRepo) DeleteSession(_ string) error {
	return nil
}

func (e *expiredSessionRepo) CreateSession(_ *sessionModel.Session) (string, error) {
	return "", errors.New("not implemented")
}

func TestSessionServiceGetUserID_ExpiredSession(t *testing.T) {
	repo := &expiredSessionRepo{}
	service := NewSessionService(repo, time.Hour)

	_, err := service.GetUserID("any-session-id")
	if err == nil {
		t.Fatalf("expected error for expired session, got nil")
	}
}
