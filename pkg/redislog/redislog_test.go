package redislog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

func TestLogOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		start   time.Time
		err     error
		args    []any
		wantMsg string
		wantLvl zapcore.Level
	}{
		{name: "debug ok", start: time.Now(), wantMsg: "redis", wantLvl: zapcore.DebugLevel},
		{name: "debug nil", start: time.Now(), err: redis.ErrNil, wantMsg: "redis", wantLvl: zapcore.DebugLevel},
		{name: "warn error", start: time.Now(), err: errors.New("boom"), wantMsg: "redis", wantLvl: zapcore.WarnLevel},
		{name: "warn slow", start: time.Now().Add(-sqllog.SlowQueryThreshold), wantMsg: "redis slow", wantLvl: zapcore.WarnLevel},
		{name: "warn slow with nil", start: time.Now().Add(-sqllog.SlowQueryThreshold), err: redis.ErrNil, wantMsg: "redis slow", wantLvl: zapcore.WarnLevel},
		{name: "with args", start: time.Now(), args: []any{ArgRedacted}, wantMsg: "redis", wantLvl: zapcore.DebugLevel},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, logs := observer.New(tt.wantLvl)
			logger := zap.New(core)

			LogOp(context.Background(), logger, "get", "GET", tt.start, tt.err, tt.args)

			require.Equal(t, 1, logs.Len())
			entry := logs.All()[0]
			require.Equal(t, tt.wantMsg, entry.Message)
			require.Equal(t, tt.wantLvl, entry.Level)
			require.Equal(t, "GET", entry.ContextMap()["cmd"])
			if len(tt.args) > 0 {
				require.Contains(t, entry.ContextMap(), "args")
			}
		})
	}
}

func TestLogOp_NilLogger(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		LogOp(context.Background(), nil, "op", "GET", time.Now(), nil, nil)
	})
}
