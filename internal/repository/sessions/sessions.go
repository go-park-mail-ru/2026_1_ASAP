package sessions

import (
	"errors"
	"sync"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
	"github.com/google/uuid"
)

type SessionRepositoryInterface interface {
	CreateSession(session *models.Session) (*models.SessionData, error)
	GetSession(sessionID string) (*models.Session, error)
	DeleteSession(sessionID string) error
}

type SessionRpository struct {
	sessions map[string]*models.Session
	mu       sync.RWMutex
}

func NewSessionRepository() *SessionRpository {
	return &SessionRpository{
		sessions: make(map[string]*models.Session),
	}
}

func (s *SessionRpository) CreateSession(session *models.Session) (*models.SessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New().String()
	s.sessions[sessionID] = session

	return &models.SessionData{
		SessionID: sessionID,
		Expire:    session.Expire,
	}, nil

}

func (s *SessionRpository) GetSession(sessionID string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("Session not found")
	}

	return session, nil
}

func (s *SessionRpository) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}
