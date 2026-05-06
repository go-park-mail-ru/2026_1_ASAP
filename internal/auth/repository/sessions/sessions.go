package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/redislog"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
)

type SessionRepository struct {
	pool   *redis.Pool
	logger *zap.Logger
	TTL    time.Duration
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

func NewSessionRepository(config config.SessionConfig, redisConfig config.RedisConfig, logger *zap.Logger) *SessionRepository {
	return &SessionRepository{
		pool:   NewRedisPool(&redisConfig),
		TTL:    config.SessionTTL,
		logger: logger,
	}
}

func (s *SessionRepository) CreateSession(ctx context.Context, session *domain.Session) (string, error) {
	conn := s.pool.Get()
	defer conn.Close()

	sessionModel := toModel(session)
	sessionValue, err := json.Marshal(sessionModel)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal session: %w", err)
	}

	key := "session:" + session.SessionID

	start := time.Now()
	ttlSec := int(time.Until(session.ExpiresAt).Seconds())
	_, err = conn.Do("SET", key, sessionValue, "EX", ttlSec)
	redislog.LogOp(ctx, s.log(ctx), "CreateSession", fmt.Sprintf("SET %s EX", key), start, err, []any{key, ttlSec, redislog.ArgRedacted})

	if err != nil {
		return "", fmt.Errorf("Failed to set session: %w", err)
	}
	return session.SessionID, nil
}

func (s *SessionRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	conn := s.pool.Get()
	defer conn.Close()

	key := "session:" + sessionID

	start := time.Now()
	sessionValue, err := redis.Bytes(conn.Do("GET", key))
	redislog.LogOp(ctx, s.log(ctx), "GetSession", fmt.Sprintf("GET %s", key), start, err, []any{key})
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

func (s *SessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	conn := s.pool.Get()
	defer conn.Close()

	key := "session:" + sessionID
	start := time.Now()
	deleted, err := conn.Do("DEL", key)
	redislog.LogOp(ctx, s.log(ctx), "DeleteSession", fmt.Sprintf("DEL %s", key), start, err, []any{key, deleted})
	if err != nil {
		return fmt.Errorf("Failed to delete session: %w", err)
	}
	if deleted == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SessionRepository) Close() {
	s.pool.Close()
}

func (s *SessionRepository) log(ctx context.Context) *zap.Logger {
	base := s.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
