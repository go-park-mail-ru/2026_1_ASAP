package stickers

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

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	stickerssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/stickers/sql"
)

func newStickerPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newStickerRepository(mock pgxmock.PgxPoolIface) *Repository {
	return &Repository{db: mock, logger: zap.NewNop()}
}

func stickerStringPtr(s string) *string {
	return &s
}

func stickerIntPtr(v int) *int {
	return &v
}

func stickerTimes() (time.Time, time.Time) {
	return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
}

func stickerRows(rows ...[]any) *pgxmock.Rows {
	out := pgxmock.NewRows([]string{
		"id", "pack_id", "file_url", "slug", "emoji", "width", "height", "sort_order", "created_at", "updated_at",
	})
	for _, row := range rows {
		out.AddRow(row...)
	}
	return out
}

func stickerRow(id, packID int64, fileURL string, slug, emoji sql.NullString, width, height sql.NullInt64, sortOrder int) []any {
	createdAt, updatedAt := stickerTimes()
	return []any{id, packID, fileURL, slug, emoji, width, height, sortOrder, createdAt, updatedAt}
}

func stickerPackRows(rows ...[]any) *pgxmock.Rows {
	out := pgxmock.NewRows([]string{
		"id", "name", "slug", "title", "thumbnail_url", "sort_order", "created_at", "updated_at",
	})
	for _, row := range rows {
		out.AddRow(row...)
	}
	return out
}

func stickerPackRow(id int64, name string, slug, title, thumbnailURL sql.NullString, sortOrder int) []any {
	createdAt, updatedAt := stickerTimes()
	return []any{id, name, slug, title, thumbnailURL, sortOrder, createdAt, updatedAt}
}

func TestRepository_GetStickerByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(pgxmock.PgxPoolIface)
		want    *domain.Sticker
		wantErr error
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetStickerByID).
					WithArgs(int64(10)).
					WillReturnRows(stickerRows(stickerRow(
						10,
						2,
						"https://cdn/sticker.webp",
						sql.NullString{String: "wave", Valid: true},
						sql.NullString{String: "👋", Valid: true},
						sql.NullInt64{Int64: 512, Valid: true},
						sql.NullInt64{Int64: 256, Valid: true},
						3,
					)))
			},
			want: &domain.Sticker{
				Id:        10,
				PackID:    2,
				FileURL:   "https://cdn/sticker.webp",
				Slug:      stickerStringPtr("wave"),
				Emoji:     stickerStringPtr("👋"),
				Width:     stickerIntPtr(512),
				Height:    stickerIntPtr(256),
				SortOrder: 3,
				CreatedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "not found",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetStickerByID).WithArgs(int64(10)).WillReturnError(pgx.ErrNoRows)
			},
			wantErr: domain.ErrStickerNotFound,
		},
		{
			name: "db error",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetStickerByID).WithArgs(int64(10)).WillReturnError(errors.New("db down"))
			},
			wantErr: errors.New("scan sticker"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock := newStickerPGMock(t)
			tt.prepare(mock)
			repo := newStickerRepository(mock)

			got, err := repo.GetStickerByID(ctx, 10)
			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrStickerNotFound) {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.ErrorContains(t, err, tt.wantErr.Error())
				}
				require.Nil(t, got)
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetStickersByIDs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ids     []int64
		prepare func(pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{name: "empty ids", ids: nil, wantLen: 0},
		{
			name: "success",
			ids:  []int64{1, 2},
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetStickersByIDs).
					WithArgs([]int64{1, 2}).
					WillReturnRows(stickerRows(
						stickerRow(1, 10, "one.webp", sql.NullString{}, sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, 1),
						stickerRow(2, 10, "two.webp", sql.NullString{}, sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, 2),
					))
			},
			wantLen: 2,
		},
		{
			name: "query error",
			ids:  []int64{1},
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetStickersByIDs).WithArgs([]int64{1}).WillReturnError(errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			ids:  []int64{1},
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "pack_id", "file_url", "slug", "emoji", "width", "height", "sort_order", "created_at", "updated_at",
				}).AddRow("bad", int64(10), "one.webp", sql.NullString{}, sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, 1, time.Now(), time.Now())
				m.ExpectQuery(stickerssql.GetStickersByIDs).WithArgs([]int64{1}).WillReturnRows(rows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock := newStickerPGMock(t)
			if tt.prepare != nil {
				tt.prepare(mock)
			}
			repo := newStickerRepository(mock)

			got, err := repo.GetStickersByIDs(ctx, tt.ids)
			if tt.wantErr {
				require.Error(t, err)
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetStickerPacks(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "empty",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetAllStickerPacks).WillReturnRows(stickerPackRows())
			},
			wantLen: 0,
		},
		{
			name: "packs with stickers",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetAllStickerPacks).
					WillReturnRows(stickerPackRows(stickerPackRow(
						7,
						"animals",
						sql.NullString{String: "animals", Valid: true},
						sql.NullString{String: "Animals", Valid: true},
						sql.NullString{String: "thumb.webp", Valid: true},
						1,
					)))
				m.ExpectQuery(stickerssql.GetStickersByPackIDs).
					WithArgs([]int64{7}).
					WillReturnRows(stickerRows(stickerRow(
						11,
						7,
						"cat.webp",
						sql.NullString{String: "cat", Valid: true},
						sql.NullString{String: "🐱", Valid: true},
						sql.NullInt64{Int64: 128, Valid: true},
						sql.NullInt64{Int64: 128, Valid: true},
						1,
					)))
			},
			wantLen: 1,
		},
		{
			name: "pack query error",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(stickerssql.GetAllStickerPacks).WillReturnError(errors.New("db down"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock := newStickerPGMock(t)
			tt.prepare(mock)
			repo := newStickerRepository(mock)

			got, err := repo.GetStickerPacks(ctx)
			if tt.wantErr {
				require.Error(t, err)
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				require.Len(t, got[0].Stickers, 1)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Close(t *testing.T) {
	mock := newStickerPGMock(t)
	repo := newStickerRepository(mock)

	require.NotPanics(t, repo.Close)
}
