package payment

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	paymentsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/repository/payment/sql"
)

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestPaymentRepository(mock pgxmock.PgxPoolIface) *PaymentRepository {
	return &PaymentRepository{db: mock, logger: zap.NewNop()}
}

func paymentStringPtr(s string) *string {
	return &s
}

func paymentRow(
	id int64,
	paymentID string,
	userID int64,
	status domain.PaymentStatus,
	amount int32,
	subscriptionDays int32,
	paymentURL sql.NullString,
	message sql.NullString,
) *pgxmock.Rows {
	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"id", "payment_id", "user_id", "status", "amount", "subscription_days", "payment_url", "message", "created_at", "updated_at",
	}).AddRow(id, paymentID, userID, string(status), amount, subscriptionDays, paymentURL, message, createdAt, updatedAt)
}

func expectedPayment(
	id int64,
	paymentID string,
	userID int64,
	status domain.PaymentStatus,
	amount int32,
	subscriptionDays int32,
	paymentURL *string,
	message *string,
) *domain.Payment {
	return &domain.Payment{
		ID:               id,
		PaymentID:        paymentID,
		UserID:           userID,
		Status:           status,
		Amount:           amount,
		SubscriptionDays: subscriptionDays,
		PaymentURL:       paymentURL,
		Message:          message,
		CreatedAt:        time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	}
}

func TestPaymentRepository_PaymentGetByID_Positive(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Payment
		name    string
		id      int64
	}{
		{
			name: "returns payment by id",
			id:   10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByID).
					WithArgs(int64(10)).
					WillReturnRows(paymentRow(
						10,
						"pay-10",
						77,
						domain.PaymentStatusPending,
						19900,
						30,
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{String: "created", Valid: true},
					))
			},
			want: expectedPayment(
				10,
				"pay-10",
				77,
				domain.PaymentStatusPending,
				19900,
				30,
				paymentStringPtr("https://payment.local/10"),
				paymentStringPtr("created"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetByID(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentGetByID_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, payment *domain.Payment, err error)
		name    string
		id      int64
	}{
		{
			name: "not found",
			id:   404,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByID).WithArgs(int64(404)).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorIs(t, err, domain.ErrPaymentNotFound)
			},
		},
		{
			name: "db error",
			id:   10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByID).WithArgs(int64(10)).WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment get by id")
			},
		},
		{
			name: "scan error",
			id:   10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "payment_id", "user_id", "status", "amount", "subscription_days", "payment_url", "message", "created_at", "updated_at",
				}).AddRow("bad-id", "pay-10", int64(77), "pending", int32(19900), int32(30), sql.NullString{}, sql.NullString{}, time.Now(), time.Now())
				m.ExpectQuery(paymentsql.GetByID).WithArgs(int64(10)).WillReturnRows(rows)
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment get by id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetByID(ctx, tt.id)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentGetByPaymentID_Positive(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		want     *domain.Payment
		name     string
		paymentID string
	}{
		{
			name:      "returns payment by external payment id with nullable fields",
			paymentID: "pay-10",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByPaymentID).
					WithArgs("pay-10").
					WillReturnRows(paymentRow(
						10,
						"pay-10",
						77,
						domain.PaymentStatusWaitingForCapture,
						19900,
						30,
						sql.NullString{Valid: false},
						sql.NullString{Valid: false},
					))
			},
			want: expectedPayment(10, "pay-10", 77, domain.PaymentStatusWaitingForCapture, 19900, 30, nil, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetByPaymentID(ctx, tt.paymentID)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentGetByPaymentID_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		assert   func(t *testing.T, payment *domain.Payment, err error)
		name     string
		paymentID string
	}{
		{
			name:      "not found",
			paymentID: "missing",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByPaymentID).WithArgs("missing").WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorIs(t, err, domain.ErrPaymentNotFound)
			},
		},
		{
			name:      "db error",
			paymentID: "pay-10",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetByPaymentID).WithArgs("pay-10").WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment get by payment id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetByPaymentID(ctx, tt.paymentID)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentGetOpenPendingByUser_Positive(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Payment
		name    string
		userID  int64
	}{
		{
			name:   "returns latest open payment",
			userID: 77,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetOpenPendingByUser).
					WithArgs(int64(77)).
					WillReturnRows(paymentRow(
						10,
						"pay-10",
						77,
						domain.PaymentStatusPending,
						19900,
						30,
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{Valid: false},
					))
			},
			want: expectedPayment(10, "pay-10", 77, domain.PaymentStatusPending, 19900, 30, paymentStringPtr("https://payment.local/10"), nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetOpenPendingByUser(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentGetOpenPendingByUser_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, payment *domain.Payment, err error)
		name    string
		userID  int64
	}{
		{
			name:   "not found",
			userID: 77,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetOpenPendingByUser).WithArgs(int64(77)).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorIs(t, err, domain.ErrPaymentNotFound)
			},
		},
		{
			name:   "db error",
			userID: 77,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.GetOpenPendingByUser).WithArgs(int64(77)).WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment get open pending by user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentGetOpenPendingByUser(ctx, tt.userID)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentCreate_Positive(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		payment *domain.Payment
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Payment
		name    string
	}{
		{
			name: "creates payment",
			payment: &domain.Payment{
				PaymentID:        "pay-10",
				UserID:           77,
				Status:           domain.PaymentStatusPending,
				Amount:           19900,
				SubscriptionDays: 30,
				PaymentURL:       paymentStringPtr("https://payment.local/10"),
				Message:          paymentStringPtr("created"),
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.Insert).
					WithArgs(
						"pay-10",
						int64(77),
						"pending",
						int32(19900),
						int32(30),
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{String: "created", Valid: true},
					).
					WillReturnRows(paymentRow(
						10,
						"pay-10",
						77,
						domain.PaymentStatusPending,
						19900,
						30,
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{String: "created", Valid: true},
					))
			},
			want: expectedPayment(10, "pay-10", 77, domain.PaymentStatusPending, 19900, 30, paymentStringPtr("https://payment.local/10"), paymentStringPtr("created")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentCreate(ctx, tt.payment)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentCreate_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		payment *domain.Payment
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, payment *domain.Payment, err error)
		name    string
	}{
		{
			name:    "nil payment",
			payment: nil,
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment is nil")
			},
		},
		{
			name: "duplicate payment",
			payment: &domain.Payment{
				PaymentID:        "pay-10",
				UserID:           77,
				Status:           domain.PaymentStatusPending,
				Amount:           19900,
				SubscriptionDays: 30,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.Insert).
					WithArgs("pay-10", int64(77), "pending", int32(19900), int32(30), sql.NullString{Valid: false}, sql.NullString{Valid: false}).
					WillReturnError(&pgconn.PgError{ConstraintName: paymentsPaymentIDKey})
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorIs(t, err, domain.ErrDuplicatePayment)
			},
		},
		{
			name: "db error",
			payment: &domain.Payment{
				PaymentID:        "pay-10",
				UserID:           77,
				Status:           domain.PaymentStatusPending,
				Amount:           19900,
				SubscriptionDays: 30,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.Insert).
					WithArgs("pay-10", int64(77), "pending", int32(19900), int32(30), sql.NullString{Valid: false}, sql.NullString{Valid: false}).
					WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment create")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentCreate(ctx, tt.payment)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentUpdate_Positive(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		payment *domain.Payment
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *domain.Payment
		name    string
	}{
		{
			name: "updates payment",
			payment: &domain.Payment{
				PaymentID:        "pay-10",
				Status:           domain.PaymentStatusSucceeded,
				Amount:           19900,
				SubscriptionDays: 30,
				PaymentURL:       paymentStringPtr("https://payment.local/10"),
				Message:          paymentStringPtr("paid"),
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.UpdateByPaymentID).
					WithArgs(
						"pay-10",
						"succeeded",
						int32(19900),
						int32(30),
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{String: "paid", Valid: true},
					).
					WillReturnRows(paymentRow(
						10,
						"pay-10",
						77,
						domain.PaymentStatusSucceeded,
						19900,
						30,
						sql.NullString{String: "https://payment.local/10", Valid: true},
						sql.NullString{String: "paid", Valid: true},
					))
			},
			want: expectedPayment(10, "pay-10", 77, domain.PaymentStatusSucceeded, 19900, 30, paymentStringPtr("https://payment.local/10"), paymentStringPtr("paid")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentUpdate(ctx, tt.payment)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentRepository_PaymentUpdate_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		payment *domain.Payment
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, payment *domain.Payment, err error)
		name    string
	}{
		{
			name:    "nil payment",
			payment: nil,
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment is nil")
			},
		},
		{
			name:    "empty payment id",
			payment: &domain.Payment{},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment id is empty")
			},
		},
		{
			name: "not found",
			payment: &domain.Payment{
				PaymentID:        "missing",
				Status:           domain.PaymentStatusCanceled,
				Amount:           19900,
				SubscriptionDays: 30,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.UpdateByPaymentID).
					WithArgs("missing", "canceled", int32(19900), int32(30), sql.NullString{Valid: false}, sql.NullString{Valid: false}).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorIs(t, err, domain.ErrPaymentNotFound)
			},
		},
		{
			name: "db error",
			payment: &domain.Payment{
				PaymentID:        "pay-10",
				Status:           domain.PaymentStatusSucceeded,
				Amount:           19900,
				SubscriptionDays: 30,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(paymentsql.UpdateByPaymentID).
					WithArgs("pay-10", "succeeded", int32(19900), int32(30), sql.NullString{Valid: false}, sql.NullString{Valid: false}).
					WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, payment *domain.Payment, err error) {
				t.Helper()
				require.Nil(t, payment)
				require.ErrorContains(t, err, "payment update")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestPaymentRepository(mock)

			got, err := repo.PaymentUpdate(ctx, tt.payment)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
