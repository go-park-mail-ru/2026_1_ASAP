package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	paymentsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/repository/payment/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

const paymentsPaymentIDKey = "payments_payment_id_key"

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type PaymentRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewPaymentRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*PaymentRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return &PaymentRepository{db: pool, logger: logger}, nil
}

func (r *PaymentRepository) Close() {
	r.db.Close()
}

func (r *PaymentRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

type paymentModel struct {
	ID               int64
	PaymentID        string
	UserID           int64
	Status           string
	Amount           int32
	SubscriptionDays int32
	PaymentURL       sql.NullString
	Message          sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (m paymentModel) toDomain() *domain.Payment {
	return &domain.Payment{
		ID:               m.ID,
		PaymentID:        m.PaymentID,
		UserID:           m.UserID,
		Status:           domain.PaymentStatus(m.Status),
		Amount:           m.Amount,
		SubscriptionDays: m.SubscriptionDays,
		PaymentURL:       null.NullStringToPtrString(m.PaymentURL),
		Message:          null.NullStringToPtrString(m.Message),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func toModel(p *domain.Payment) paymentModel {
	if p == nil {
		return paymentModel{}
	}
	return paymentModel{
		ID:               p.ID,
		PaymentID:        p.PaymentID,
		UserID:           p.UserID,
		Status:           string(p.Status),
		Amount:           p.Amount,
		SubscriptionDays: p.SubscriptionDays,
		PaymentURL:       null.StringPtrToNullString(p.PaymentURL),
		Message:          null.StringPtrToNullString(p.Message),
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func (r *PaymentRepository) PaymentGetByID(ctx context.Context, id int64) (*domain.Payment, error) {
	q := paymentsql.GetByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, id)

	var m paymentModel
	err := row.Scan(
		&m.ID,
		&m.PaymentID,
		&m.UserID,
		&m.Status,
		&m.Amount,
		&m.SubscriptionDays,
		&m.PaymentURL,
		&m.Message,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "PaymentGetByID", q, start, err, []any{id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("payment get by id: %w", err)
	}

	return m.toDomain(), nil
}

func (r *PaymentRepository) PaymentGetByPaymentID(ctx context.Context, paymentID string) (*domain.Payment, error) {
	q := paymentsql.GetByPaymentID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, paymentID)

	var m paymentModel
	err := row.Scan(
		&m.ID,
		&m.PaymentID,
		&m.UserID,
		&m.Status,
		&m.Amount,
		&m.SubscriptionDays,
		&m.PaymentURL,
		&m.Message,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "PaymentGetByPaymentID", q, start, err, []any{paymentID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("payment get by payment id: %w", err)
	}

	return m.toDomain(), nil
}

func (r *PaymentRepository) PaymentCreate(ctx context.Context, p *domain.Payment) (*domain.Payment, error) {
	if p == nil {
		return nil, errors.New("payment is nil")
	}

	m := toModel(p)
	q := paymentsql.Insert
	start := time.Now()
	err := r.db.QueryRow(ctx, q,
		m.PaymentID,
		m.UserID,
		m.Status,
		m.Amount,
		m.SubscriptionDays,
		m.PaymentURL,
		m.Message,
	).Scan(
		&m.ID,
		&m.PaymentID,
		&m.UserID,
		&m.Status,
		&m.Amount,
		&m.SubscriptionDays,
		&m.PaymentURL,
		&m.Message,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "PaymentCreate", q, start, err, []any{
		m.PaymentID, m.UserID, m.Status, m.Amount, m.SubscriptionDays, m.PaymentURL, m.Message,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == paymentsPaymentIDKey {
			return nil, domain.ErrDuplicatePayment
		}
		return nil, fmt.Errorf("payment create: %w", err)
	}

	return m.toDomain(), nil
}

func (r *PaymentRepository) PaymentUpdate(ctx context.Context, p *domain.Payment) (*domain.Payment, error) {
	if p == nil {
		return nil, errors.New("payment is nil")
	}
	if p.PaymentID == "" {
		return nil, errors.New("payment id is empty")
	}

	m := toModel(p)
	q := paymentsql.UpdateByPaymentID
	start := time.Now()
	err := r.db.QueryRow(ctx, q,
		m.PaymentID,
		m.Status,
		m.Amount,
		m.SubscriptionDays,
		m.PaymentURL,
		m.Message,
	).Scan(
		&m.ID,
		&m.PaymentID,
		&m.UserID,
		&m.Status,
		&m.Amount,
		&m.SubscriptionDays,
		&m.PaymentURL,
		&m.Message,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "PaymentUpdate", q, start, err, []any{
		m.PaymentID, m.Status, m.Amount, m.SubscriptionDays, m.PaymentURL, m.Message,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("payment update: %w", err)
	}

	return m.toDomain(), nil
}
