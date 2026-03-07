package session

import (
	"errors"
	"time"

	sessionModel "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
	"github.com/google/uuid"
)

type SessionServiceInterface interface {
	CreateSession(userID uuid.UUID) (*sessionModel.SessionData, error)
	GetUserID(sessionID string) (uuid.UUID, error)
	DeleteSession(sessionID string) error
}

type SessionService struct {
	sessionRepository sessions.SessionRepositoryInterface
	sessionTTL        time.Duration
}

func NewSessionService(repository sessions.SessionRepositoryInterface, sessionTTL time.Duration) *SessionService {
	return &SessionService{
		sessionRepository: repository,
		sessionTTL:        sessionTTL,
	}
}

func (sessionService *SessionService) CreateSession(userID uuid.UUID) (*sessionModel.SessionData, error) {
	session := &sessionModel.Session{
		UserID: userID,
		Expire: time.Now().Add(sessionService.sessionTTL),
	}
	return sessionService.sessionRepository.CreateSession(session)
}

func (sessionService *SessionService) GetUserID(sessionID string) (uuid.UUID, error) {

	session, err := sessionService.sessionRepository.GetSession(sessionID)
	if err != nil {
		return uuid.Nil, err
	}

	if time.Now().After(session.Expire) {
		sessionService.sessionRepository.DeleteSession(sessionID)
		return uuid.Nil, errors.New("session expired")
	}

	return session.UserID, nil
}

func (sessionService *SessionService) DeleteSession(sessionID string) error {
	return sessionService.sessionRepository.DeleteSession(sessionID)
}
func (sessionService *SessionService) TTL() time.Duration {
	return sessionService.sessionTTL
}
