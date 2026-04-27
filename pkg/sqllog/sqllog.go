package sqllog

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var SlowQueryThreshold = 500 * time.Millisecond

const ArgRedacted = "[redacted]"

func LogQuery(ctx context.Context, logger *zap.Logger, op, query string, start time.Time, err error, args []any) {
	if logger == nil {
		return
	}
	d := time.Since(start)
	fields := []zap.Field{
		zap.String("db_op", op),
		zap.String("query", oneLine(query)),
		zap.Duration("duration", d),
	}
	if len(args) > 0 {
		fields = append(fields, zap.Any("args", args))
	}
	lg := logger.With(fields...)
	switch {
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		lg.Warn("db", zap.Error(err))
	case d >= SlowQueryThreshold:
		if err != nil {
			lg.Warn("db slow", zap.Error(err))
		} else {
			lg.Warn("db slow")
		}
	default:
		if err != nil {
			lg.Debug("db", zap.NamedError("err", err))
		} else {
			lg.Debug("db")
		}
	}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
