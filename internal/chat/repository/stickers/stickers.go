package stickers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	stickerssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/stickers/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

type dbPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

type Repository struct {
	db     dbPool
	logger *zap.Logger
}

func NewRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &Repository{db: pool, logger: logger}, nil
}

func (r *Repository) GetStickerPacks(ctx context.Context) ([]domain.StickerPack, error) {
	start := time.Now()
	rows, err := r.db.Query(ctx, stickerssql.GetAllStickerPacks)
	sqllog.LogQuery(ctx, r.log(ctx), "GetStickerPacks.packs", stickerssql.GetAllStickerPacks, start, err, nil)
	if err != nil {
		return nil, fmt.Errorf("query sticker packs: %w", err)
	}
	defer rows.Close()

	packs := make([]domain.StickerPack, 0)
	packIDs := make([]int64, 0)
	packByID := make(map[int64]int)
	for rows.Next() {
		model := &StickerPackModel{}
		if err = rows.Scan(
			&model.Id,
			&model.Name,
			&model.Slug,
			&model.Title,
			&model.ThumbnailURL,
			&model.SortOrder,
			&model.CreatedAt,
			&model.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sticker pack: %w", err)
		}
		pack := packToDomain(model)
		packByID[pack.Id] = len(packs)
		packIDs = append(packIDs, pack.Id)
		packs = append(packs, *pack)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sticker packs: %w", err)
	}
	if len(packIDs) == 0 {
		return packs, nil
	}

	stickers, err := r.getStickersByPackIDs(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for _, sticker := range stickers {
		if idx, ok := packByID[sticker.PackID]; ok {
			packs[idx].Stickers = append(packs[idx].Stickers, sticker)
		}
	}
	return packs, nil
}

func (r *Repository) GetStickerByID(ctx context.Context, stickerID int64) (*domain.Sticker, error) {
	start := time.Now()
	model := &StickerModel{}
	err := r.db.QueryRow(ctx, stickerssql.GetStickerByID, stickerID).Scan(
		&model.Id,
		&model.PackID,
		&model.FileURL,
		&model.Slug,
		&model.Emoji,
		&model.Width,
		&model.Height,
		&model.SortOrder,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetStickerByID", stickerssql.GetStickerByID, start, err, []any{stickerID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStickerNotFound
		}
		return nil, fmt.Errorf("scan sticker: %w", err)
	}
	return stickerToDomain(model), nil
}

func (r *Repository) GetStickersByIDs(ctx context.Context, stickerIDs []int64) (map[int64]domain.Sticker, error) {
	if len(stickerIDs) == 0 {
		return map[int64]domain.Sticker{}, nil
	}
	start := time.Now()
	rows, err := r.db.Query(ctx, stickerssql.GetStickersByIDs, stickerIDs)
	sqllog.LogQuery(ctx, r.log(ctx), "GetStickersByIDs", stickerssql.GetStickersByIDs, start, err, []any{stickerIDs})
	if err != nil {
		return nil, fmt.Errorf("query stickers by ids: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]domain.Sticker, len(stickerIDs))
	for rows.Next() {
		model := &StickerModel{}
		if err = scanSticker(rows, model); err != nil {
			return nil, err
		}
		sticker := stickerToDomain(model)
		out[sticker.Id] = *sticker
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stickers by ids: %w", err)
	}
	return out, nil
}

func (r *Repository) getStickersByPackIDs(ctx context.Context, packIDs []int64) ([]domain.Sticker, error) {
	start := time.Now()
	rows, err := r.db.Query(ctx, stickerssql.GetStickersByPackIDs, packIDs)
	sqllog.LogQuery(ctx, r.log(ctx), "GetStickersByPackIDs", stickerssql.GetStickersByPackIDs, start, err, []any{packIDs})
	if err != nil {
		return nil, fmt.Errorf("query stickers by pack ids: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Sticker, 0)
	for rows.Next() {
		model := &StickerModel{}
		if err = scanSticker(rows, model); err != nil {
			return nil, err
		}
		out = append(out, *stickerToDomain(model))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stickers by pack ids: %w", err)
	}
	return out, nil
}

func scanSticker(rows pgx.Rows, model *StickerModel) error {
	if err := rows.Scan(
		&model.Id,
		&model.PackID,
		&model.FileURL,
		&model.Slug,
		&model.Emoji,
		&model.Width,
		&model.Height,
		&model.SortOrder,
		&model.CreatedAt,
		&model.UpdatedAt,
	); err != nil {
		return fmt.Errorf("scan sticker: %w", err)
	}
	return nil
}

func (r *Repository) Close() {
	r.db.Close()
}

func (r *Repository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
