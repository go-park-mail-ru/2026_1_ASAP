package sessions

import (
	"github.com/google/uuid"
	"sync"
	"errors"
	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
)

type SessionRepositoryInterface interface {
    CreateSession(session *models.Session) (string, error)
    GetUserID(sessionID string) (uuid.UUID, error)
    DeleteSession(sessionID string) error
}

type SessionRpository struct {
	sessions map[string]*models.Session
	mu sync.RWMutex
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

func (s *SessionRpository) GetUserID(sessionID string) (uuid.UUID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return uuid.Nil, errors.New("userID not found")
	}

	return session.UserID, nil
}

func (s *SessionRpository) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}