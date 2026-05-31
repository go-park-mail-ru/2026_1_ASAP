package sqllog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		start   time.Time
		err     error
		args    []any
		wantMsg string
		wantLvl zapcore.Level
	}{
		{name: "debug ok", start: time.Now(), wantMsg: "db", wantLvl: zapcore.DebugLevel},
		{name: "debug no rows", start: time.Now(), err: pgx.ErrNoRows, wantMsg: "db", wantLvl: zapcore.DebugLevel},
		{name: "warn error", start: time.Now(), err: errors.New("boom"), wantMsg: "db", wantLvl: zapcore.WarnLevel},
		{name: "warn slow", start: time.Now().Add(-SlowQueryThreshold), wantMsg: "db slow", wantLvl: zapcore.WarnLevel},
		{name: "warn slow with no rows", start: time.Now().Add(-SlowQueryThreshold), err: pgx.ErrNoRows, wantMsg: "db slow", wantLvl: zapcore.WarnLevel},
		{name: "with args", start: time.Now(), args: []any{ArgRedacted}, wantMsg: "db", wantLvl: zapcore.DebugLevel},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core, logs := observer.New(tt.wantLvl)
			logger := zap.New(core)

			LogQuery(context.Background(), logger, "select", "select\n  1", tt.start, tt.err, tt.args)

			require.Equal(t, 1, logs.Len())
			entry := logs.All()[0]
			require.Equal(t, tt.wantMsg, entry.Message)
			require.Equal(t, tt.wantLvl, entry.Level)
			require.Equal(t, "select 1", entry.ContextMap()["query"])
			if len(tt.args) > 0 {
				require.Contains(t, entry.ContextMap(), "args")
			}
		})
	}
}

func TestLogQuery_NilLogger(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		LogQuery(context.Background(), nil, "op", "select 1", time.Now(), nil, nil)
	})
}

func TestOneLine(t *testing.T) {
	t.Parallel()

	require.Equal(t, "select * from users", oneLine(" select\n\t*   from users "))
}
