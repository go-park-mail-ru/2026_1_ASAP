package complaint

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/complaint"
	complaintsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/repository/complaint/sql"
	//"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrString(s string) *string {
	return &s
}

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestComplaintRepository(mock pgxmock.PgxPoolIface) *ComplaintRepository {
	return &ComplaintRepository{db: mock, logger: zap.NewNop()}
}

func newComplaintRow(t *testing.T, id int64, status, complaintType, feedbackName, feedbackEmail, body string, userID sql.NullInt64, attachmentURL sql.NullString) *pgxmock.Rows {
	t.Helper()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
	}).AddRow(id, status, complaintType, feedbackName, feedbackEmail, body, userID, attachmentURL, now, now)
}

func TestPositiveComplaintRepository_GetAllComplaints(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    []domain.Complaint
		name    string
	}{
		{
			name: "Get all complaints successfully",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
				}).AddRow(
					int64(1), "new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true}, now, now,
				).AddRow(
					int64(2), "in_progress", "product", "Jane Smith", "jane@example.com", "Product suggestion", sql.NullInt64{Valid: false}, sql.NullString{Valid: false}, now, now,
				)
				m.ExpectQuery(complaintsql.GetAll).WillReturnRows(rows)
			},
			want: []domain.Complaint{
				{
					ID:            1,
					Status:        "new",
					Type:          "bug",
					FeedBackName:  "John Doe",
					FeedBackEmail: "john@example.com",
					Body:          "Bug description",
					UserID:        ptrInt64(100),
					AttachmentURL: ptrString("file.jpg"),
					CreatedAt:     now,
					UpdatedAt:     now,
				},
				{
					ID:            2,
					Status:        "in_progress",
					Type:          "product",
					FeedBackName:  "Jane Smith",
					FeedBackEmail: "jane@example.com",
					Body:          "Product suggestion",
					UserID:        nil,
					AttachmentURL: nil,
					CreatedAt:     now,
					UpdatedAt:     now,
				},
			},
		},
		{
			name: "Get all complaints empty result",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
				})
				m.ExpectQuery(complaintsql.GetAll).WillReturnRows(rows)
			},
			want: []domain.Complaint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			got, err := repo.GetAllComplaints(ctx)
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				require.Equal(t, tt.want[i].ID, got[i].ID)
				require.Equal(t, tt.want[i].Status, got[i].Status)
				require.Equal(t, tt.want[i].Type, got[i].Type)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_GetAllComplaints(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
		wantErr  string
	}{
		{
			name: "Query error",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.GetAll).WillReturnError(errors.New("db error"))
			},
			wantErr: "get complaints by user id",
		},
		{
			name: "Scan row error",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
				}).AddRow(
					"invalid", "new", "bug", "John", "john@example.com", "body", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{}, time.Now(), time.Now(),
				)
				m.ExpectQuery(complaintsql.GetAll).WillReturnRows(rows)
			},
			wantErr: "scan complaint row",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.GetAllComplaints(ctx)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveComplaintRepository_Create(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		complaint domain.Complaint
		prepare   func(t *testing.T, m pgxmock.PgxPoolIface, complaint domain.Complaint)
		want      domain.Complaint
		name      string
	}{
		{
			name: "Create complaint successfully",
			complaint: domain.Complaint{
				Type:          "bug",
				FeedBackName:  "John Doe",
				FeedBackEmail: "john@example.com",
				Body:          "Bug description",
				UserID:        ptrInt64(100),
				AttachmentURL: ptrString("file.jpg"),
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, c domain.Complaint) {
				rows := newComplaintRow(t, int64(1), "new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true})
				m.ExpectQuery(complaintsql.InsertComplaint).WithArgs(
					"new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true},
				).WillReturnRows(rows)
			},
			want: domain.Complaint{
				ID:            1,
				Status:        "new",
				Type:          "bug",
				FeedBackName:  "John Doe",
				FeedBackEmail: "john@example.com",
				Body:          "Bug description",
				UserID:        ptrInt64(100),
				AttachmentURL: ptrString("file.jpg"),
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		{
			name: "Create complaint without user and attachment",
			complaint: domain.Complaint{
				Type:          "product",
				FeedBackName:  "Jane Smith",
				FeedBackEmail: "jane@example.com",
				Body:          "Product suggestion",
				UserID:        nil,
				AttachmentURL: nil,
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, c domain.Complaint) {
				rows := newComplaintRow(t, int64(2), "new", "product", "Jane Smith", "jane@example.com", "Product suggestion", sql.NullInt64{Valid: false}, sql.NullString{Valid: false})
				m.ExpectQuery(complaintsql.InsertComplaint).WithArgs(
					"new", "product", "Jane Smith", "jane@example.com", "Product suggestion", sql.NullInt64{Valid: false}, sql.NullString{Valid: false},
				).WillReturnRows(rows)
			},
			want: domain.Complaint{
				ID:            2,
				Status:        "new",
				Type:          "product",
				FeedBackName:  "Jane Smith",
				FeedBackEmail: "jane@example.com",
				Body:          "Product suggestion",
				UserID:        nil,
				AttachmentURL: nil,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock, tt.complaint)
			repo := newTestComplaintRepository(mock)

			got, err := repo.Create(ctx, tt.complaint)
			require.NoError(t, err)
			require.Equal(t, tt.want.ID, got.ID)
			require.Equal(t, tt.want.Status, got.Status)
			require.Equal(t, tt.want.Type, got.Type)
			require.Equal(t, tt.want.FeedBackName, got.FeedBackName)
			require.Equal(t, tt.want.Body, got.Body)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_Create(t *testing.T) {
	ctx := context.Background()

	complaint := domain.Complaint{
		Type:          "bug",
		FeedBackName:  "John Doe",
		FeedBackEmail: "john@example.com",
		Body:          "Bug description",
	}

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
		wantErr  string
	}{
		{
			name: "Insert error",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.InsertComplaint).WithArgs(
					"new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Valid: false}, sql.NullString{Valid: false},
				).WillReturnError(errors.New("insert failed"))
			},
			wantErr: "create complaint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.Create(ctx, complaint)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveComplaintRepository_GetComplaintByID(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		want     domain.Complaint
		name     string
		complaintID int64
	}{
		{
			name:        "Get complaint by id successfully",
			complaintID: 1,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := newComplaintRow(t, int64(1), "new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true})
				m.ExpectQuery(complaintsql.GetComplaintByID).WithArgs(int64(1)).WillReturnRows(rows)
			},
			want: domain.Complaint{
				ID:            1,
				Status:        "new",
				Type:          "bug",
				FeedBackName:  "John Doe",
				FeedBackEmail: "john@example.com",
				Body:          "Bug description",
				UserID:        ptrInt64(100),
				AttachmentURL: ptrString("file.jpg"),
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			got, err := repo.GetComplaintByID(ctx, tt.complaintID)
			require.NoError(t, err)
			require.Equal(t, tt.want.ID, got.ID)
			require.Equal(t, tt.want.Status, got.Status)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_GetComplaintByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
		complaintID int64
		wantErr  string
	}{
		{
			name:        "Complaint not found",
			complaintID: 999,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.GetComplaintByID).WithArgs(int64(999)).WillReturnError(pgx.ErrNoRows)
			},
			wantErr: "complaint not found",
		},
		{
			name:        "Query error",
			complaintID: 1,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.GetComplaintByID).WithArgs(int64(1)).WillReturnError(errors.New("db error"))
			},
			wantErr: "get complaint by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.GetComplaintByID(ctx, tt.complaintID)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveComplaintRepository_GetComplaintsByUserID(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    []domain.Complaint
		name    string
		userID  int64
	}{
		{
			name:   "Get complaints by user id successfully",
			userID: 100,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
				}).AddRow(
					int64(1), "new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true}, now, now,
				).AddRow(
					int64(2), "in_progress", "product", "John Doe", "john@example.com", "Product suggestion", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{Valid: false}, now, now,
				)
				m.ExpectQuery(complaintsql.GetComplaintsByUserID).WithArgs(int64(100)).WillReturnRows(rows)
			},
			want: []domain.Complaint{
				{
					ID:            1,
					Status:        "new",
					Type:          "bug",
					FeedBackName:  "John Doe",
					FeedBackEmail: "john@example.com",
					Body:          "Bug description",
					UserID:        ptrInt64(100),
					AttachmentURL: ptrString("file.jpg"),
					CreatedAt:     now,
					UpdatedAt:     now,
				},
				{
					ID:            2,
					Status:        "in_progress",
					Type:          "product",
					FeedBackName:  "John Doe",
					FeedBackEmail: "john@example.com",
					Body:          "Product suggestion",
					UserID:        ptrInt64(100),
					AttachmentURL: nil,
					CreatedAt:     now,
					UpdatedAt:     now,
				},
			},
		},
		{
			name:   "Get complaints by user id empty result",
			userID: 200,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "status", "type", "feedback_name", "feedback_email", "body", "user_id", "attachment_url", "created_at", "updated_at",
				})
				m.ExpectQuery(complaintsql.GetComplaintsByUserID).WithArgs(int64(200)).WillReturnRows(rows)
			},
			want: []domain.Complaint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			got, err := repo.GetComplaintsByUserID(ctx, tt.userID)
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_GetComplaintsByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		userID  int64
		wantErr string
	}{
		{
			name:   "Query error",
			userID: 100,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.GetComplaintsByUserID).WithArgs(int64(100)).WillReturnError(errors.New("db error"))
			},
			wantErr: "get complaints by user id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.GetComplaintsByUserID(ctx, tt.userID)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveComplaintRepository_UploadAttachmentURL(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare       func(t *testing.T, m pgxmock.PgxPoolIface)
		want          domain.Complaint
		name          string
		complaintID   int64
		attachmentURL string
	}{
		{
			name:          "Upload attachment URL successfully",
			complaintID:   1,
			attachmentURL: "new_file.jpg",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := newComplaintRow(t, int64(1), "new", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "new_file.jpg", Valid: true})
				m.ExpectQuery(complaintsql.UploadAttachmentURL).WithArgs(int64(1), "new_file.jpg").WillReturnRows(rows)
			},
			want: domain.Complaint{
				ID:            1,
				Status:        "new",
				Type:          "bug",
				FeedBackName:  "John Doe",
				FeedBackEmail: "john@example.com",
				Body:          "Bug description",
				UserID:        ptrInt64(100),
				AttachmentURL: ptrString("new_file.jpg"),
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			got, err := repo.UploadAttachmentURL(ctx, tt.complaintID, tt.attachmentURL)
			require.NoError(t, err)
			require.Equal(t, tt.want.AttachmentURL, got.AttachmentURL)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_UploadAttachmentURL(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare       func(t *testing.T, m pgxmock.PgxPoolIface)
		name          string
		complaintID   int64
		attachmentURL string
		wantErr       string
	}{
		{
			name:          "Complaint not found",
			complaintID:   999,
			attachmentURL: "file.jpg",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.UploadAttachmentURL).WithArgs(int64(999), "file.jpg").WillReturnError(pgx.ErrNoRows)
			},
			wantErr: "complaint not found",
		},
		{
			name:          "Query error",
			complaintID:   1,
			attachmentURL: "file.jpg",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.UploadAttachmentURL).WithArgs(int64(1), "file.jpg").WillReturnError(errors.New("db error"))
			},
			wantErr: "upload complaint attachment url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.UploadAttachmentURL(ctx, tt.complaintID, tt.attachmentURL)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveComplaintRepository_UpdateComplaint(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare      func(t *testing.T, m pgxmock.PgxPoolIface)
		want         domain.Complaint
		name         string
		complaintID  int64
		status       domain.ComplaintStatus
	}{
		{
			name:        "Update complaint status successfully",
			complaintID: 1,
			status:      "in_progress",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := newComplaintRow(t, int64(1), "in_progress", "bug", "John Doe", "john@example.com", "Bug description", sql.NullInt64{Int64: 100, Valid: true}, sql.NullString{String: "file.jpg", Valid: true})
				m.ExpectQuery(complaintsql.UpdateComplaintStatus).WithArgs(int64(1), "in_progress").WillReturnRows(rows)
			},
			want: domain.Complaint{
				ID:            1,
				Status:        "in_progress",
				Type:          "bug",
				FeedBackName:  "John Doe",
				FeedBackEmail: "john@example.com",
				Body:          "Bug description",
				UserID:        ptrInt64(100),
				AttachmentURL: ptrString("file.jpg"),
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			got, err := repo.UpdateComplaint(ctx, tt.complaintID, tt.status)
			require.NoError(t, err)
			require.Equal(t, tt.want.Status, got.Status)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeComplaintRepository_UpdateComplaint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare     func(t *testing.T, m pgxmock.PgxPoolIface)
		name        string
		complaintID int64
		status      domain.ComplaintStatus
		wantErr     string
	}{
		{
			name:        "Complaint not found",
			complaintID: 999,
			status:      "closed",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.UpdateComplaintStatus).WithArgs(int64(999), "closed").WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrNotFound.Error(),
		},
		{
			name:        "Query error",
			complaintID: 1,
			status:      "closed",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(complaintsql.UpdateComplaintStatus).WithArgs(int64(1), "closed").WillReturnError(errors.New("db error"))
			},
			wantErr: "update complaint status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestComplaintRepository(mock)

			_, err := repo.UpdateComplaint(ctx, tt.complaintID, tt.status)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestComplaintRepository_Close(t *testing.T) {
	mock := newPGMock(t)
	repo := newTestComplaintRepository(mock)

	// Close не должен паниковать
	repo.Close()
}

func TestComplaintRepository_Log(t *testing.T) {
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
			repo := &ComplaintRepository{
				logger: tt.logger,
			}
			got := repo.log(context.Background())
			if tt.expectNotNil {
				require.NotNil(t, got)
			}
		})
	}
}

func TestToDomain_WithNullValues(t *testing.T) {
	now := time.Now()
	model := ComplaintModel{
		ID:            1,
		Status:        "new",
		Type:          "bug",
		FeedBackName:  "John Doe",
		FeedBackEmail: "john@example.com",
		Body:          "Description",
		UserID:        sql.NullInt64{Int64: 100, Valid: true},
		AttachmentURL: sql.NullString{String: "file.jpg", Valid: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result := toDomain(model)
	require.Equal(t, int64(1), result.ID)
	require.Equal(t, "new", result.Status)
	require.Equal(t, domain.ComplaintType("bug"), result.Type)
	require.Equal(t, "John Doe", result.FeedBackName)
	require.Equal(t, "john@example.com", result.FeedBackEmail)
	require.Equal(t, "Description", result.Body)
	require.Equal(t, int64(100), *result.UserID)
	require.Equal(t, "file.jpg", *result.AttachmentURL)
}

func TestToDomain_WithNilValues(t *testing.T) {
	now := time.Now()
	model := ComplaintModel{
		ID:            2,
		Status:        "new",
		Type:          "product",
		FeedBackName:  "Jane Smith",
		FeedBackEmail: "jane@example.com",
		Body:          "Suggestion",
		UserID:        sql.NullInt64{Valid: false},
		AttachmentURL: sql.NullString{Valid: false},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result := toDomain(model)
	require.Nil(t, result.UserID)
	require.Nil(t, result.AttachmentURL)
}

func TestToModel(t *testing.T) {
	now := time.Now()
	complaint := domain.Complaint{
		ID:            1,
		Status:        "new",
		Type:          "bug",
		FeedBackName:  "John Doe",
		FeedBackEmail: "john@example.com",
		Body:          "Description",
		UserID:        ptrInt64(100),
		AttachmentURL: ptrString("file.jpg"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result := toModel(complaint)
	require.Equal(t, int64(1), result.ID)
	require.Equal(t, "new", result.Status)
	require.Equal(t, "bug", result.Type)
	require.Equal(t, "John Doe", result.FeedBackName)
	require.Equal(t, "john@example.com", result.FeedBackEmail)
	require.Equal(t, "Description", result.Body)
	require.Equal(t, sql.NullInt64{Int64: 100, Valid: true}, result.UserID)
	require.Equal(t, sql.NullString{String: "file.jpg", Valid: true}, result.AttachmentURL)
}