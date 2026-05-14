package analytic

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/analytic"
	analyticsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/repository/analytic/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type AnalyticRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewAnalyticRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*AnalyticRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	return &AnalyticRepository{db: pool, logger: logger}, nil
}

func (r *AnalyticRepository) GetUserAnalytic(ctx context.Context, userID int64) (domain.ComplaintAnalytic, error) {
	q := analyticsql.GetUserAnalytic
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userID)

	var response domain.ComplaintAnalytic
	err := row.Scan(
		&response.CountStatus.CountStatusOpened,
		&response.CountStatus.CountStatusInWork,
		&response.CountStatus.CountStatusClosed,
		&response.CountType.CountBug,
		&response.CountType.CountUpgrade,
		&response.CountType.CountProduct,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserAnalytic", q, start, err, []any{userID})
	if err != nil {
		return domain.ComplaintAnalytic{}, fmt.Errorf("get user complaint analytic: %w", err)
	}

	return response, nil
}

func (r *AnalyticRepository) Close() {
	r.db.Close()
}

func (r *AnalyticRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
