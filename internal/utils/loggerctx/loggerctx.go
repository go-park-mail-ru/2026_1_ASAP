package loggerctx

import (
	"context"

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
	if logger == nil {
		return zap.NewNop()
	}
	if ctx == nil {
		return logger
	}
	return logger
}
