package session

import (
	"errors"
	"time"

	"github.com/google/uuid"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	sessionDTO "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
)

type SessionRepository interface {
	CreateSession(session *domain.Session) (string, error)
	GetSession(sessionID string) (*domain.Session, error)
	DeleteSession(sessionID string) error
}

type SessionService struct {
	sessionRepository SessionRepository
	sessionTTL        time.Duration
}

func NewSessionService(repository SessionRepository, sessionTTL time.Duration) *SessionService {
	return &SessionService{
		sessionRepository: repository,
		sessionTTL:        sessionTTL,
	}
}

func (sessionService *SessionService) CreateSession(userID uuid.UUID) (*sessionDTO.SessionDTO, error) {
	session := &domain.Session{
		UserID: userID,
		Expire: time.Now().Add(sessionService.sessionTTL),
	}
	sessionID, err := sessionService.sessionRepository.CreateSession(session)
	if err != nil {
		return nil, err
	}

	return &sessionDTO.SessionDTO{
		SessionID: sessionID,
		Expire:    session.Expire,
	}, nil
}

func (sessionService *SessionService) GetUserID(sessionID string) (uuid.UUID, error) {

	session, err := sessionService.sessionRepository.GetSession(sessionID)
	if err != nil {
		return uuid.Nil, err
	}

	if time.Now().After(session.Expire) {
		if err := sessionService.sessionRepository.DeleteSession(sessionID); err != nil {
			return uuid.Nil, errors.New("session expired and failed to delete session")
		}
		return uuid.Nil, errors.New("session expired")
	}

	return session.UserID, nil
}

func (sessionService *SessionService) DeleteSession(sessionID string) error {
	return sessionService.sessionRepository.DeleteSession(sessionID)
}
