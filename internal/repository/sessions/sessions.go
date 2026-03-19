package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

type SessionRepository struct {
	pool *redis.Pool
	TTL  time.Duration
}

func NewRedisPool(config *config.RedisConfig) *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(
				"tcp",
				config.ServerInfo(),
				redis.DialPassword(config.Password),
				redis.DialDatabase(config.Database),
				redis.DialConnectTimeout(2*time.Second),
				redis.DialReadTimeout(2*time.Second),
				redis.DialWriteTimeout(2*time.Second),
			)
		},
		TestOnBorrowContext: func(ctx context.Context, c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewSessionRepository(config config.SessionConfig, redisConfig config.RedisConfig) *SessionRepository {
	return &SessionRepository{
		pool: NewRedisPool(&redisConfig),
		TTL:  config.SessionTTL,
	}
}

func (s SessionRepository) CreateSession(ctx context.Context, session *domain.Session) (string, error) {
	conn := s.pool.Get()
	defer conn.Close()

	sessionModel := toModel(session)
	sessionValue, err := json.Marshal(sessionModel)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal session: %w", err)
	}

	key := "session:" + session.SessionID

	_, err = conn.Do("SET", key, sessionValue, "EX", int(s.TTL.Seconds()))

	if err != nil {
		return "", fmt.Errorf("Failed to set session: %w", err)
	}
	return session.SessionID, nil
}

func (s SessionRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	conn := s.pool.Get()
	defer conn.Close()

	key := "session:" + sessionID

	sessionValue, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("Failed to get session: %w", err)
	}

	var session SessionModel
	if err := json.Unmarshal(sessionValue, &session); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal session: %w", err)
	}

	return toDomain(&session), nil
}

func (s SessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	conn := s.pool.Get()
	defer conn.Close()

	key := "session:" + sessionID
	deleted, err := conn.Do("DEL", key)
	if err != nil {
		return fmt.Errorf("Failed to delete session: %w", err)
	}
	if deleted == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Cтарье
type DepricatedSessionRpository struct {
	sessions map[string]*domain.DepricatedSession
	mu       sync.RWMutex
}

func NewDepricatedSessionRepository() *DepricatedSessionRpository {
	return &DepricatedSessionRpository{
		sessions: make(map[string]*domain.DepricatedSession),
	}
}

func (s *DepricatedSessionRpository) GetSession(sessionID string) (*domain.DepricatedSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("Session not found")
	}

	return session, nil
}

func (s *DepricatedSessionRpository) CreateSession(session *domain.DepricatedSession) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New().String()
	s.sessions[sessionID] = session

	return sessionID, nil

}

func (s *DepricatedSessionRpository) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}
