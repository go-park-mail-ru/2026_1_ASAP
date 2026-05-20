package online

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/sessions"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/redislog"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

const onlineKeyPrefix = "presence:online:"

// OnlineTTL — без ping/активности ключ истекает, пользователь offline в REST.
const OnlineTTL = 90 * time.Second

type RedisRepository struct {
	pool   *redis.Pool
	logger *zap.Logger
}

func NewRedisRepository(cfg config.RedisConfig, logger *zap.Logger) *RedisRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedisRepository{
		pool:   sessions.NewRedisPool(&cfg),
		logger: logger,
	}
}

func onlineKey(userID int64) string {
	return onlineKeyPrefix + strconv.FormatInt(userID, 10)
}

func (r *RedisRepository) setOnlineKey(ctx context.Context, userID int64) error {
	conn := r.pool.Get()
	defer conn.Close()

	key := onlineKey(userID)
	ttlSec := int(OnlineTTL.Seconds())
	start := time.Now()
	_, err := conn.Do("SET", key, "1", "EX", ttlSec)
	redislog.LogOp(ctx, r.log(ctx), "SetOnline", "SET "+key+" EX", start, err, []any{key, ttlSec})
	if err != nil {
		return fmt.Errorf("presence set online: %w", err)
	}
	return nil
}

func (r *RedisRepository) SetOnline(ctx context.Context, userID int64) error {
	return r.setOnlineKey(ctx, userID)
}

func (r *RedisRepository) TouchOnline(ctx context.Context, userID int64) error {
	return r.setOnlineKey(ctx, userID)
}

func (r *RedisRepository) SetOffline(ctx context.Context, userID int64) error {
	conn := r.pool.Get()
	defer conn.Close()

	key := onlineKey(userID)
	start := time.Now()
	_, err := conn.Do("DEL", key)
	redislog.LogOp(ctx, r.log(ctx), "SetOffline", "DEL "+key, start, err, []any{key})
	if err != nil {
		return fmt.Errorf("presence set offline: %w", err)
	}
	return nil
}

func (r *RedisRepository) IsOnline(ctx context.Context, userID int64) (bool, error) {
	conn := r.pool.Get()
	defer conn.Close()

	key := onlineKey(userID)
	start := time.Now()
	n, err := redis.Int(conn.Do("EXISTS", key))
	redislog.LogOp(ctx, r.log(ctx), "IsOnline", "EXISTS "+key, start, err, []any{key})
	if err != nil {
		return false, fmt.Errorf("presence is online: %w", err)
	}
	return n > 0, nil
}

func (r *RedisRepository) log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, r.logger)
}
