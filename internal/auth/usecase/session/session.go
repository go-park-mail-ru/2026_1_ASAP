package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sessionDTO "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	utils "github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/csrf"
)

//go:generate mockgen -source=session.go -destination=mock/session_mock.go -package=mock
type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.Session) (string, error)
	GetSession(ctx context.Context, sessionID string) (*domain.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type SessionService struct {
	sessionRepository SessionRepository
	sessionTTL        time.Duration
}

func (sessionService *SessionService) GetCSRFToken(ctx context.Context, sessionID string) (string, error) {
	session, err := sessionService.sessionRepository.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
		return "", domain.ErrExpired
	}

	if session.CSRFToken == "" {
		return "", domain.ErrCSRFNotFound
	}

	if session.CSRFExpiresAt.Before(time.Now()) {
		return "", domain.ErrCSRFExpired
	}

	return session.CSRFToken, nil
}

func (sessionService *SessionService) SetCSRFToken(ctx context.Context, sessionID string, token string) error {
	session, err := sessionService.sessionRepository.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
		return domain.ErrExpired
	}
	session.CSRFToken = token

	session.CSRFExpiresAt = time.Now().Add(1 * time.Hour)
	session.CSRFToken = token

	_, err = sessionService.sessionRepository.CreateSession(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func NewSessionService(repository SessionRepository, sessionTTL time.Duration) *SessionService {
	return &SessionService{
		sessionRepository: repository,
		sessionTTL:        sessionTTL,
	}
}

func (sessionService *SessionService) CreateSession(ctx context.Context, userID int64) (*sessionDTO.SessionDTO, error) {
	token, err := utils.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session := &domain.Session{
		SessionID:     uuid.New().String(),
		UserID:        userID,
		CSRFToken:     token,
		CSRFExpiresAt: time.Now().Add(1 * time.Hour),
		ExpiresAt:     time.Now().Add(sessionService.sessionTTL),
	}
	sessionID, err := sessionService.sessionRepository.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &sessionDTO.SessionDTO{
		SessionID: sessionID,
		Expire:    session.ExpiresAt,
		CSRFToken: session.CSRFToken,
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

	if session.ExpiresAt.Before(time.Now()) {
		return 0, domain.ErrExpired
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
