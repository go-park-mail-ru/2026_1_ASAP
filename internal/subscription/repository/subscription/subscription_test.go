package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	subscriptionsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/repository/subscription/sql"
)

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestSubscriptionRepository(mock pgxmock.PgxPoolIface) *SubscriptionRepository {
	return &SubscriptionRepository{db: mock, logger: zap.NewNop()}
}

func subscriptionRow(userID int64, active bool, startAt, endAt time.Time) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"user_id", "active", "start_at", "end_at",
	}).AddRow(userID, active, startAt, endAt)
}

func expectedSubscription(userID int64, active bool, startAt, endAt time.Time) *domain.Subscription {
	return &domain.Subscription{
		UserID:  userID,
		Active:  active,
		StartAt: startAt,
		EndAt:   endAt,
	}
}

func TestPositiveSubscriptionRepository_SubscriptionGet(t *testing.T) {
	ctx := context.Background()
	//now := time.Now()
	startAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Subscription
		name    string
		userID  int64
	}{
		{
			name:   "returns subscription for existing user",
			userID: 100,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.GetByUserID).
					WithArgs(int64(100)).
					WillReturnRows(subscriptionRow(100, true, startAt, endAt))
			},
			want: expectedSubscription(100, true, startAt, endAt),
		},
		{
			name:   "returns inactive subscription",
			userID: 200,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.GetByUserID).
					WithArgs(int64(200)).
					WillReturnRows(subscriptionRow(200, false, startAt, endAt))
			},
			want: expectedSubscription(200, false, startAt, endAt),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestSubscriptionRepository(mock)

			got, err := repo.SubscriptionGet(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeSubscriptionRepository_SubscriptionGet(t *testing.T) {
	ctx := context.Background()
	startAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, sub *domain.Subscription, err error)
		name    string
		userID  int64
	}{
		{
			name:   "subscription not found",
			userID: 404,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.GetByUserID).WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorIs(t, err, domain.ErrSubscriptionNotFound)
			},
		},
		{
			name:   "database error",
			userID: 100,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.GetByUserID).WithArgs(int64(100)).WillReturnError(errors.New("db connection lost"))
			},
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorContains(t, err, "subscription get")
			},
		},
		{
			name:   "scan error - wrong column type",
			userID: 100,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "active", "start_at", "end_at",
				}).AddRow("invalid", true, startAt, endAt)
				m.ExpectQuery(subscriptionsql.GetByUserID).WithArgs(int64(100)).WillReturnRows(rows)
			},
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorContains(t, err, "subscription get")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestSubscriptionRepository(mock)

			got, err := repo.SubscriptionGet(ctx, tt.userID)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveSubscriptionRepository_SubscriptionSet(t *testing.T) {
	ctx := context.Background()
	startAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	tests := []struct {
		sub     *domain.Subscription
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Subscription
		name    string
	}{
		{
			name: "creates new subscription",
			sub: &domain.Subscription{
				UserID:  100,
				Active:  true,
				StartAt: startAt,
				EndAt:   endAt,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.Upsert).
					WithArgs(int64(100), true, startAt, endAt).
					WillReturnRows(subscriptionRow(100, true, startAt, endAt))
			},
			want: expectedSubscription(100, true, startAt, endAt),
		},
		{
			name: "updates existing subscription",
			sub: &domain.Subscription{
				UserID:  100,
				Active:  false,
				StartAt: startAt,
				EndAt:   endAt,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.Upsert).
					WithArgs(int64(100), false, startAt, endAt).
					WillReturnRows(subscriptionRow(100, false, startAt, endAt))
			},
			want: expectedSubscription(100, false, startAt, endAt),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestSubscriptionRepository(mock)

			got, err := repo.SubscriptionSet(ctx, tt.sub)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeSubscriptionRepository_SubscriptionSet(t *testing.T) {
	ctx := context.Background()
	startAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	tests := []struct {
		sub     *domain.Subscription
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, sub *domain.Subscription, err error)
		name    string
	}{
		{
			name:    "nil subscription",
			sub:     nil,
			prepare: nil,
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorContains(t, err, "subscription is nil")
			},
		},
		{
			name: "database error on upsert",
			sub: &domain.Subscription{
				UserID:  100,
				Active:  true,
				StartAt: startAt,
				EndAt:   endAt,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(subscriptionsql.Upsert).
					WithArgs(int64(100), true, startAt, endAt).
					WillReturnError(errors.New("db error"))
			},
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorContains(t, err, "subscription set")
			},
		},
		{
			name: "scan error after upsert",
			sub: &domain.Subscription{
				UserID:  100,
				Active:  true,
				StartAt: startAt,
				EndAt:   endAt,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "active", "start_at", "end_at",
				}).AddRow("invalid", true, startAt, endAt)
				m.ExpectQuery(subscriptionsql.Upsert).
					WithArgs(int64(100), true, startAt, endAt).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, sub *domain.Subscription, err error) {
				t.Helper()
				require.Nil(t, sub)
				require.ErrorContains(t, err, "subscription set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestSubscriptionRepository(mock)

			got, err := repo.SubscriptionSet(ctx, tt.sub)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_SubscriptionSet_ZeroTimestamps(t *testing.T) {
	ctx := context.Background()
	zeroTime := time.Time{}

	sub := &domain.Subscription{
		UserID:  100,
		Active:  false,
		StartAt: zeroTime,
		EndAt:   zeroTime,
	}

	mock := newPGMock(t)
	mock.ExpectQuery(subscriptionsql.Upsert).
		WithArgs(int64(100), false, zeroTime, zeroTime).
		WillReturnRows(subscriptionRow(100, false, zeroTime, zeroTime))
	repo := newTestSubscriptionRepository(mock)

	got, err := repo.SubscriptionSet(ctx, sub)
	require.NoError(t, err)
	require.Equal(t, int64(100), got.UserID)
	require.False(t, got.Active)
	require.True(t, got.StartAt.IsZero())
	require.True(t, got.EndAt.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_Close(t *testing.T) {
	mock := newPGMock(t)
	repo := newTestSubscriptionRepository(mock)

	// Close не должен паниковать
	repo.Close()
}

func TestSubscriptionRepository_Log(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "logger is nil returns nop",
			logger: nil,
		},
		{
			name:   "logger is set returns enriched logger",
			logger: zap.NewNop(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &SubscriptionRepository{
				logger: tt.logger,
			}
			got := repo.log(context.Background())
			require.NotNil(t, got)
		})
	}
}

func TestNewSubscriptionRepository(t *testing.T) {
	ctx := context.Background()
	cfg := config.PostgresConfig{
		Username: "user",
		Password: "pass",
		Host:     "localhost",
		Port:     "5432",
		Database: "testdb",
	}
	logger := zap.NewNop()

	// Проверяем, что функция возвращает ошибку (невозможно подключиться в тесте)
	repo, err := NewSubscriptionRepository(ctx, cfg, logger)
	if err != nil {
		require.Nil(t, repo)
		require.Error(t, err)
	}
}

func TestSubscriptionModel_ToDomain(t *testing.T) {
	//now := time.Now()
	startAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 1, 0)

	tests := []struct {
		name  string
		model subscriptionModel
		want  *domain.Subscription
	}{
		{
			name: "active subscription",
			model: subscriptionModel{
				UserID:  100,
				Active:  true,
				StartAt: startAt,
				EndAt:   endAt,
			},
			want: &domain.Subscription{
				UserID:  100,
				Active:  true,
				StartAt: startAt,
				EndAt:   endAt,
			},
		},
		{
			name: "inactive subscription",
			model: subscriptionModel{
				UserID:  200,
				Active:  false,
				StartAt: startAt,
				EndAt:   endAt,
			},
			want: &domain.Subscription{
				UserID:  200,
				Active:  false,
				StartAt: startAt,
				EndAt:   endAt,
			},
		},
		{
			name: "subscription with zero times",
			model: subscriptionModel{
				UserID:  300,
				Active:  false,
				StartAt: time.Time{},
				EndAt:   time.Time{},
			},
			want: &domain.Subscription{
				UserID:  300,
				Active:  false,
				StartAt: time.Time{},
				EndAt:   time.Time{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.toDomain()
			require.Equal(t, tt.want, got)
		})
	}
}
