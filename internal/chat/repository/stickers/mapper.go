package stickers

import (
	"database/sql"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

func packToDomain(model *StickerPackModel) *domain.StickerPack {
	return &domain.StickerPack{
		Id:           model.Id,
		Name:         model.Name,
		Slug:         null.NullStringToPtrString(model.Slug),
		Title:        null.NullStringToPtrString(model.Title),
		ThumbnailURL: null.NullStringToPtrString(model.ThumbnailURL),
		SortOrder:    model.SortOrder,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func stickerToDomain(model *StickerModel) *domain.Sticker {
	return &domain.Sticker{
		Id:        model.Id,
		PackID:    model.PackID,
		FileURL:   model.FileURL,
		Slug:      null.NullStringToPtrString(model.Slug),
		Emoji:     null.NullStringToPtrString(model.Emoji),
		Width:     nullInt64ToIntPtr(model.Width),
		Height:    nullInt64ToIntPtr(model.Height),
		SortOrder: model.SortOrder,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func nullInt64ToIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	out := int(v.Int64)
	return &out
}
