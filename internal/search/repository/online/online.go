package online

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/sessions"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/redislog"
)

const onlineKeyPrefix = "presence:online:"

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

func (r *RedisRepository) FilterOnline(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	keys := make([]interface{}, 0, len(userIDs))
	ids := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		keys = append(keys, onlineKey(id))
		ids = append(ids, id)
	}
	if len(keys) == 0 {
		return out, nil
	}

	conn := r.pool.Get()
	defer conn.Close()

	start := time.Now()
	replies, err := redis.Values(conn.Do("MGET", keys...))
	redislog.LogOp(ctx, r.log(ctx), "FilterOnline", "MGET", start, err, []any{len(keys)})

	if err != nil {
		return nil, fmt.Errorf("presence filter online: %w", err)
	}

	for i, id := range ids {
		if i >= len(replies) {
			break
		}
		_, err := redis.String(replies[i], nil)
		out[id] = err == nil
	}
	return out, nil
}

func (r *RedisRepository) log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, r.logger)
}
