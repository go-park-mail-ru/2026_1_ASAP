package complaint

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
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/complaint"
	complaintsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/complaint/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/sqllog"
)

type dbPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type ComplaintRepository struct {
	db     dbPool
	logger *zap.Logger
}

type ComplaintModel struct {
	ID            int64
	Status        string
	Type          string
	FeedBackName  string
	FeedBackEmail string
	Body          string
	UserID        sql.NullInt64
	AttachmentURL sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewComplaintRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*ComplaintRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return &ComplaintRepository{db: pool, logger: logger}, nil
}

func (r *ComplaintRepository) Create(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
	model := toModel(complaint)
	if model.Status == "" {
		model.Status = "new"
	}

	q := complaintsql.InsertComplaint
	start := time.Now()
	err := r.db.QueryRow(ctx, q,
		model.Status,
		model.Type,
		model.FeedBackName,
		model.FeedBackEmail,
		model.Body,
		model.UserID,
		model.AttachmentURL,
	).Scan(
		&model.ID,
		&model.Status,
		&model.Type,
		&model.FeedBackName,
		&model.FeedBackEmail,
		&model.Body,
		&model.UserID,
		&model.AttachmentURL,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "CreateComplaint", q, start, err, []any{
		model.Status, model.Type, model.FeedBackName, model.FeedBackEmail, model.Body, model.UserID, model.AttachmentURL,
	})
	if err != nil {
		return domain.Complaint{}, fmt.Errorf("create complaint: %w", err)
	}

	return toDomain(model), nil
}

func (r *ComplaintRepository) GetComplaintByID(ctx context.Context, id int64) (domain.Complaint, error) {
	q := complaintsql.GetComplaintByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, id)

	model := ComplaintModel{}
	err := row.Scan(
		&model.ID,
		&model.Status,
		&model.Type,
		&model.FeedBackName,
		&model.FeedBackEmail,
		&model.Body,
		&model.UserID,
		&model.AttachmentURL,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetComplaintByID", q, start, err, []any{id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Complaint{}, fmt.Errorf("complaint not found")
		}
		return domain.Complaint{}, fmt.Errorf("get complaint by id: %w", err)
	}

	return toDomain(model), nil
}

func (r *ComplaintRepository) GetComplaintsByUserID(ctx context.Context, userID int64) ([]domain.Complaint, error) {
	q := complaintsql.GetComplaintsByUserID
	start := time.Now()
	rows, err := r.db.Query(ctx, q, userID)
	sqllog.LogQuery(ctx, r.log(ctx), "GetComplaintsByUserID", q, start, err, []any{userID})
	if err != nil {
		return nil, fmt.Errorf("get complaints by user id: %w", err)
	}
	defer rows.Close()

	complaints := make([]domain.Complaint, 0)
	for rows.Next() {
		model := ComplaintModel{}
		if err := rows.Scan(
			&model.ID,
			&model.Status,
			&model.Type,
			&model.FeedBackName,
			&model.FeedBackEmail,
			&model.Body,
			&model.UserID,
			&model.AttachmentURL,
			&model.CreatedAt,
			&model.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan complaint row: %w", err)
		}

		complaints = append(complaints, toDomain(model))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate complaint rows: %w", err)
	}

	return complaints, nil
}

func (r *ComplaintRepository) UploadAttachmentURL(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error) {
	q := complaintsql.UploadAttachmentURL
	start := time.Now()
	row := r.db.QueryRow(ctx, q, complaintID, attachmentURL)

	model := ComplaintModel{}
	err := row.Scan(
		&model.ID,
		&model.Status,
		&model.Type,
		&model.FeedBackName,
		&model.FeedBackEmail,
		&model.Body,
		&model.UserID,
		&model.AttachmentURL,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadAttachmentURL", q, start, err, []any{complaintID, attachmentURL})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Complaint{}, fmt.Errorf("complaint not found")
		}
		return domain.Complaint{}, fmt.Errorf("upload complaint attachment url: %w", err)
	}

	return toDomain(model), nil
}

func (r *ComplaintRepository) UpdateComplaint(ctx context.Context, complaintID int64, status domain.ComplaintType) (domain.Complaint, error) {
	q := complaintsql.UpdateComplaintStatus
	start := time.Now()
	row := r.db.QueryRow(ctx, q, complaintID, string(status))

	model := ComplaintModel{}
	err := row.Scan(
		&model.ID,
		&model.Status,
		&model.Type,
		&model.FeedBackName,
		&model.FeedBackEmail,
		&model.Body,
		&model.UserID,
		&model.AttachmentURL,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UpdateComplaintStatus", q, start, err, []any{complaintID, status})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Complaint{}, fmt.Errorf("complaint not found")
		}
		return domain.Complaint{}, fmt.Errorf("update complaint status: %w", err)
	}

	return toDomain(model), nil
}

func (r *ComplaintRepository) Close() {
	r.db.Close()
}

func (r *ComplaintRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

func toDomain(model ComplaintModel) domain.Complaint {
	return domain.Complaint{
		ID:            model.ID,
		Status:        model.Status,
		Type:          domain.ComplaintType(model.Type),
		FeedBackName:  model.FeedBackName,
		FeedBackEmail: model.FeedBackEmail,
		Body:          model.Body,
		UserID:        null.NullInt64ToPtrInt64(model.UserID),
		AttachmentURL: null.NullStringToPtrString(model.AttachmentURL),
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

func toModel(complaint domain.Complaint) ComplaintModel {
	return ComplaintModel{
		ID:            complaint.ID,
		Status:        complaint.Status,
		Type:          string(complaint.Type),
		FeedBackName:  complaint.FeedBackName,
		FeedBackEmail: complaint.FeedBackEmail,
		Body:          complaint.Body,
		UserID:        null.PtrInt64ToNullInt64(complaint.UserID),
		AttachmentURL: null.StringPtrToNullString(complaint.AttachmentURL),
		CreatedAt:     complaint.CreatedAt,
		UpdatedAt:     complaint.UpdatedAt,
	}
}
