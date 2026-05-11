package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	subscriptionsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/repository/subscription/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type SubscriptionRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewSubscriptionRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*SubscriptionRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return &SubscriptionRepository{db: pool, logger: logger}, nil
}

func (r *SubscriptionRepository) SubscriptionGet(ctx context.Context, userID int64) (*domain.Subscription, error) {
	q := subscriptionsql.GetByUserID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userID)

	var m subscriptionModel
	err := row.Scan(&m.UserID, &m.Active, &m.StartAt, &m.EndAt)
	sqllog.LogQuery(ctx, r.log(ctx), "SubscriptionGet", q, start, err, []any{userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("subscription get: %w", err)
	}

	return m.toDomain(), nil
}

func (r *SubscriptionRepository) SubscriptionSet(ctx context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
	if sub == nil {
		return nil, errors.New("subscription is nil")
	}

	q := subscriptionsql.Upsert
	start := time.Now()
	row := r.db.QueryRow(ctx, q, sub.UserID, sub.Active, sub.StartAt, sub.EndAt)

	var m subscriptionModel
	err := row.Scan(&m.UserID, &m.Active, &m.StartAt, &m.EndAt)
	sqllog.LogQuery(ctx, r.log(ctx), "SubscriptionSet", q, start, err, []any{sub.UserID, sub.Active, sub.StartAt, sub.EndAt})
	if err != nil {
		return nil, fmt.Errorf("subscription set: %w", err)
	}

	return m.toDomain(), nil
}

func (r *SubscriptionRepository) Close() {
	r.db.Close()
}

func (r *SubscriptionRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

type subscriptionModel struct {
	UserID  int64
	Active  bool
	StartAt time.Time
	EndAt   time.Time
}

func (m subscriptionModel) toDomain() *domain.Subscription {
	return &domain.Subscription{
		UserID:  m.UserID,
		Active:  m.Active,
		StartAt: m.StartAt,
		EndAt:   m.EndAt,
	}
}
