package loggerctx

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestLoggerContext(t *testing.T) {
	t.Run("with nil logger keeps context", func(t *testing.T) {
		ctx := context.Background()
		if got := With(ctx, nil); got != ctx {
			t.Fatalf("With nil logger returned different context")
		}
	})

	t.Run("from returns stored logger", func(t *testing.T) {
		logger := zap.NewNop()
		ctx := With(context.Background(), logger)
		if got := From(ctx); got != logger {
			t.Fatalf("From() returned unexpected logger")
		}
	})

	t.Run("from returns nop for missing logger", func(t *testing.T) {
		if got := From(context.Background()); got == nil {
			t.Fatalf("From() returned nil")
		}
	})
}

func TestEnrichLoggerFromContext(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		if got := EnrichLoggerFromContext(context.Background(), nil); got == nil {
			t.Fatalf("EnrichLoggerFromContext() returned nil")
		}
	})

	t.Run("nil context", func(t *testing.T) {
		logger := zap.NewNop()
		if got := EnrichLoggerFromContext(nil, logger); got != logger {
			t.Fatalf("EnrichLoggerFromContext() returned unexpected logger")
		}
	})

	t.Run("logger with context", func(t *testing.T) {
		logger := zap.NewNop()
		if got := EnrichLoggerFromContext(context.Background(), logger); got != logger {
			t.Fatalf("EnrichLoggerFromContext() returned unexpected logger")
		}
	})
}
