package loggerctx

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"go.uber.org/zap"
)

type loggerKey struct{}

func With(ctx context.Context, l *zap.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

func From(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.NewNop()
}

func EnrichLoggerFromContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	reqID, ok := middleware.RequestIDFromContext(ctx)
	if !ok {
		reqID = "unknown_request_id"
	}
	return logger.With(zap.String("request_id", reqID))
}
