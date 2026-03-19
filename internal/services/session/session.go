package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	sessionDTO "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/google/uuid"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.Session) (string, error)
	GetSession(ctx context.Context, sessionID string) (*domain.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
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

func (sessionService *SessionService) CreateSession(ctx context.Context, userID int64) (*sessionDTO.SessionDTO, error) {
	session := &domain.Session{
		SessionID: uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionService.sessionTTL),
	}
	sessionID, err := sessionService.sessionRepository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &sessionDTO.SessionDTO{
		SessionID: sessionID,
		Expire:    session.ExpiresAt,
	}, nil
}

func (sessionService *SessionService) GetUserID(ctx context.Context, sessionID string) (int64, error) {

	session, err := sessionService.sessionRepository.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, domain.ErrNotFound
		}

		return 0, fmt.Errorf("failed to get session: %w", err)
	}

	return session.UserID, nil
}

func (sessionService *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	err := sessionService.sessionRepository.DeleteSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// Старый удалить
type DepricatedSessionRepository interface {
	CreateSession(session *domain.DepricatedSession) (string, error)
	GetSession(sessionID string) (*domain.DepricatedSession, error)
	DeleteSession(sessionID string) error
}

type DepricatedSessionService struct {
	sessionRepository DepricatedSessionRepository
	sessionTTL        time.Duration
}

func NewDepricatedSessionService(repository DepricatedSessionRepository, sessionTTL time.Duration) *DepricatedSessionService {
	return &DepricatedSessionService{
		sessionRepository: repository,
		sessionTTL:        sessionTTL,
	}
}

func (sessionService *DepricatedSessionService) CreateSession(userID uuid.UUID) (*sessionDTO.SessionDTO, error) {
	session := &domain.DepricatedSession{
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

func (sessionService *DepricatedSessionService) GetUserID(sessionID string) (uuid.UUID, error) {

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

func (sessionService *DepricatedSessionService) DeleteSession(sessionID string) error {
	return sessionService.sessionRepository.DeleteSession(sessionID)
}
