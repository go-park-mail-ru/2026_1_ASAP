package sessions

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
)

type SessionRpository struct {
	sessions map[string]*models.Session
	mu       sync.RWMutex
}

func NewSessionRepository() *SessionRpository {
	return &SessionRpository{
		sessions: make(map[string]*models.Session),
	}
}

func (s *SessionRpository) CreateSession(session *models.Session) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New().String()
	s.sessions[sessionID] = session

	return sessionID, nil

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
