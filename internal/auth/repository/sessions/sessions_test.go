package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRedisConn struct {
	doFunc      func(commandName string, args ...interface{}) (reply interface{}, err error)
	lastCommand string
	lastArgs    []interface{}
}

func (f *fakeRedisConn) Close() error { return nil }
func (f *fakeRedisConn) Err() error   { return nil }
func (f *fakeRedisConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	if f.doFunc == nil {
		return nil, nil
	}
	return f.doFunc(commandName, args...)
}
func (f *fakeRedisConn) Send(commandName string, args ...interface{}) error {
	f.lastCommand = commandName
	f.lastArgs = args
	return nil
}
func (f *fakeRedisConn) Flush() error                                       { return nil }
func (f *fakeRedisConn) Receive() (reply interface{}, err error) {
	if f.doFunc == nil {
		return nil, nil
	}
	return f.doFunc(f.lastCommand, f.lastArgs...)
}

func newTestSessionRepository(conn redis.Conn) *SessionRepository {
	return &SessionRepository{
		pool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return conn, nil
			},
		},
		logger: zap.NewNop(),
	}
}

func newSession() *domain.Session {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	return &domain.Session{
		SessionID:     "sid-123",
		UserID:        42,
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
		CSRFToken:     "csrf-123",
		CSRFExpiresAt: now.Add(30 * time.Minute),
	}
}

func TestSessionRepository_CreateSession_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		session *domain.Session
		name    string
	}{
		{
			name:    "stores session in redis",
			session: newSession(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeRedisConn{
				doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
					if len(args) >= 4 {
						require.Equal(t, "session:sid-123", args[0])
						require.Equal(t, "EX", args[2])
						require.IsType(t, int(0), args[3])
					}
					return "OK", nil
				},
			}
			repo := newTestSessionRepository(conn)

			got, err := repo.CreateSession(ctx, tt.session)
			require.NoError(t, err)
			require.Equal(t, "sid-123", got)
		})
	}
}

func TestSessionRepository_CreateSession_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		session    *domain.Session
		prepare    func() redis.Conn
		assert     func(t *testing.T, got string, err error)
		name       string
	}{
		{
			name:    "set error",
			session: newSession(),
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return nil, errors.New("redis down")
					},
				}
			},
			assert: func(t *testing.T, got string, err error) {
				t.Helper()
				require.Empty(t, got)
				require.Error(t, err)
				require.ErrorContains(t, err, "Failed to set session")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestSessionRepository(tt.prepare())
			got, err := repo.CreateSession(ctx, tt.session)
			tt.assert(t, got, err)
		})
	}
}

func TestSessionRepository_GetSession_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		sessionID string
	}{
		{
			name:      "returns session",
			sessionID: "sid-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(toModel(newSession()))
			require.NoError(t, err)

			conn := &fakeRedisConn{
				doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
					if len(args) >= 1 {
						require.Equal(t, "session:sid-123", args[0])
					}
					return payload, nil
				},
			}
			repo := newTestSessionRepository(conn)

			got, err := repo.GetSession(ctx, tt.sessionID)
			require.NoError(t, err)
			require.Equal(t, int64(42), got.UserID)
			require.Equal(t, "sid-123", got.SessionID)
			require.Equal(t, "csrf-123", got.CSRFToken)
		})
	}
}

func TestSessionRepository_GetSession_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare   func() redis.Conn
		assert    func(t *testing.T, got *domain.Session, err error)
		name      string
		sessionID string
	}{
		{
			name:      "not found",
			sessionID: "missing",
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return nil, redis.ErrNil
					},
				}
			},
			assert: func(t *testing.T, got *domain.Session, err error) {
				t.Helper()
				require.Nil(t, got)
				require.ErrorIs(t, err, domain.ErrNotFound)
			},
		},
		{
			name:      "redis get error",
			sessionID: "sid-123",
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return nil, errors.New("redis down")
					},
				}
			},
			assert: func(t *testing.T, got *domain.Session, err error) {
				t.Helper()
				require.Nil(t, got)
				require.Error(t, err)
				require.ErrorContains(t, err, "Failed to get session")
			},
		},
		{
			name:      "invalid json",
			sessionID: "sid-123",
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return []byte("{bad json"), nil
					},
				}
			},
			assert: func(t *testing.T, got *domain.Session, err error) {
				t.Helper()
				require.Nil(t, got)
				require.Error(t, err)
				require.ErrorContains(t, err, "Failed to unmarshal session")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestSessionRepository(tt.prepare())
			got, err := repo.GetSession(ctx, tt.sessionID)
			tt.assert(t, got, err)
		})
	}
}

func TestSessionRepository_DeleteSession_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		sessionID string
	}{
		{
			name:      "deletes session",
			sessionID: "sid-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeRedisConn{
				doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
					if len(args) >= 1 {
						require.Equal(t, "session:sid-123", args[0])
					}
					return 1, nil
				},
			}
			repo := newTestSessionRepository(conn)

			err := repo.DeleteSession(ctx, tt.sessionID)
			require.NoError(t, err)
		})
	}
}

func TestSessionRepository_DeleteSession_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare   func() redis.Conn
		assert    func(t *testing.T, err error)
		name      string
		sessionID string
	}{
		{
			name:      "not found",
			sessionID: "missing",
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return 0, nil
					},
				}
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrNotFound)
			},
		},
		{
			name:      "redis del error",
			sessionID: "sid-123",
			prepare: func() redis.Conn {
				return &fakeRedisConn{
					doFunc: func(commandName string, args ...interface{}) (interface{}, error) {
						return nil, errors.New("redis down")
					},
				}
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "Failed to delete session")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestSessionRepository(tt.prepare())
			err := repo.DeleteSession(ctx, tt.sessionID)
			tt.assert(t, err)
		})
	}
}
