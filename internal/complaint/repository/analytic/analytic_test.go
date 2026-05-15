package analytic

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/analytic"
	analyticsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/repository/analytic/sql"
)

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestAnalyticRepository(mock pgxmock.PgxPoolIface) *AnalyticRepository {
	return &AnalyticRepository{db: mock, logger: zap.NewNop()}
}

func TestPositiveAnalyticRepository_GetUserAnalytic(t *testing.T) {
	ctx := context.Background()

	type args struct {
		userID int64
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    domain.ComplaintAnalytic
		name    string
		args    args
	}{
		{
			name: "Get user analytic successfully with all data",
			args: args{
				userID: 100,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"count_status_opened", "count_status_in_work", "count_status_closed",
					"count_bug", "count_upgrade", "count_product",
				}).AddRow(
					int64(5), int64(3), int64(2),
					int64(4), int64(3), int64(3),
				)
				m.ExpectQuery(analyticsql.GetUserAnalytic).WithArgs(int64(100)).WillReturnRows(rows)
			},
			want: domain.ComplaintAnalytic{
				CountStatus: domain.CountStatus{
					CountStatusOpened: 5,
					CountStatusInWork: 3,
					CountStatusClosed: 2,
				},
				CountType: domain.CountType{
					CountBug:     4,
					CountUpgrade: 3,
					CountProduct: 3,
				},
			},
		},
		{
			name: "Get user analytic with zeros",
			args: args{
				userID: 200,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"count_status_opened", "count_status_in_work", "count_status_closed",
					"count_bug", "count_upgrade", "count_product",
				}).AddRow(
					int64(0), int64(0), int64(0),
					int64(0), int64(0), int64(0),
				)
				m.ExpectQuery(analyticsql.GetUserAnalytic).WithArgs(int64(200)).WillReturnRows(rows)
			},
			want: domain.ComplaintAnalytic{
				CountStatus: domain.CountStatus{
					CountStatusOpened: 0,
					CountStatusInWork: 0,
					CountStatusClosed: 0,
				},
				CountType: domain.CountType{
					CountBug:     0,
					CountUpgrade: 0,
					CountProduct: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestAnalyticRepository(mock)

			got, err := repo.GetUserAnalytic(ctx, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want.CountStatus.CountStatusOpened, got.CountStatus.CountStatusOpened)
			require.Equal(t, tt.want.CountStatus.CountStatusInWork, got.CountStatus.CountStatusInWork)
			require.Equal(t, tt.want.CountStatus.CountStatusClosed, got.CountStatus.CountStatusClosed)
			require.Equal(t, tt.want.CountType.CountBug, got.CountType.CountBug)
			require.Equal(t, tt.want.CountType.CountUpgrade, got.CountType.CountUpgrade)
			require.Equal(t, tt.want.CountType.CountProduct, got.CountType.CountProduct)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeAnalyticRepository_GetUserAnalytic(t *testing.T) {
	ctx := context.Background()

	type args struct {
		userID int64
	}

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
		args     args
		wantErr  string
	}{
		{
			name: "Query error",
			args: args{
				userID: 100,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(analyticsql.GetUserAnalytic).WithArgs(int64(100)).WillReturnError(errors.New("db connection error"))
			},
			wantErr: "get user complaint analytic",
		},
		{
			name: "No rows - user has no complaints",
			args: args{
				userID: 999,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(analyticsql.GetUserAnalytic).WithArgs(int64(999)).WillReturnError(pgx.ErrNoRows)
			},
			wantErr: "get user complaint analytic",
		},
		{
			name: "Scan error - wrong column type",
			args: args{
				userID: 100,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"count_status_opened", "count_status_in_work", "count_status_closed",
					"count_bug", "count_upgrade", "count_product",
				}).AddRow(
					"invalid", int64(3), int64(2),
					int64(4), int64(3), int64(3),
				)
				m.ExpectQuery(analyticsql.GetUserAnalytic).WithArgs(int64(100)).WillReturnRows(rows)
			},
			wantErr: "get user complaint analytic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestAnalyticRepository(mock)

			_, err := repo.GetUserAnalytic(ctx, tt.args.userID)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAnalyticRepository_Close(t *testing.T) {
	mock := newPGMock(t)
	repo := newTestAnalyticRepository(mock)

	// Close не должен паниковать
	repo.Close()
}

func TestAnalyticRepository_Log(t *testing.T) {
	tests := []struct {
		name         string
		logger       *zap.Logger
		expectNotNil bool
	}{
		{
			name:         "logger is nil returns nop",
			logger:       nil,
			expectNotNil: true,
		},
		{
			name:         "logger is set returns enriched logger",
			logger:       zap.NewNop(),
			expectNotNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &AnalyticRepository{
				logger: tt.logger,
			}
			got := repo.log(context.Background())
			if tt.expectNotNil {
				require.NotNil(t, got)
			}
		})
	}
}