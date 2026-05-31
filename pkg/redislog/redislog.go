package redislog

import (
	"context"
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

// ArgRedacted вместо value в args (тело сессии и т.п.).
const ArgRedacted = "[redacted]"

func LogOp(ctx context.Context, logger *zap.Logger, op, cmd string, start time.Time, err error, args []any) {
	if logger == nil {
		return
	}
	d := time.Since(start)
	fields := []zap.Field{
		zap.String("redis_op", op),
		zap.String("cmd", cmd),
		zap.Duration("duration", d),
	}
	if len(args) > 0 {
		fields = append(fields, zap.Any("args", args))
	}
	lg := logger.With(fields...)
	switch {
	case err != nil && !errors.Is(err, redis.ErrNil):
		lg.Warn("redis", zap.Error(err))
	case d >= sqllog.SlowQueryThreshold:
		if err != nil {
			lg.Warn("redis slow", zap.NamedError("err", err))
		} else {
			lg.Warn("redis slow")
		}
	default:
		if err != nil {
			lg.Debug("redis", zap.NamedError("err", err))
		} else {
			lg.Debug("redis")
		}
	}
}
