package stickers

import (
	"context"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/sticker"
)

type Repository interface {
	GetStickerPacks(ctx context.Context) ([]domain.StickerPack, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetStickerPacks(ctx context.Context) (*dto.ResponseGetStickerPacks, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("stickers repository is nil")
	}
	packs, err := s.repo.GetStickerPacks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.StickerPackDTO, 0, len(packs))
	for _, pack := range packs {
		title := pack.Name
		if pack.Title != nil && *pack.Title != "" {
			title = *pack.Title
		}
		item := dto.StickerPackDTO{
			ID:           pack.Id,
			Name:         pack.Name,
			Slug:         pack.Slug,
			Title:        title,
			ThumbnailURL: pack.ThumbnailURL,
			Stickers:     make([]dto.StickerDTO, 0, len(pack.Stickers)),
		}
		for _, sticker := range pack.Stickers {
			item.Stickers = append(item.Stickers, toDTO(sticker))
		}
		out = append(out, item)
	}
	return &dto.ResponseGetStickerPacks{Packs: out}, nil
}

func toDTO(sticker domain.Sticker) dto.StickerDTO {
	return dto.StickerDTO{
		ID:      sticker.Id,
		PackID:  sticker.PackID,
		Slug:    sticker.Slug,
		Emoji:   sticker.Emoji,
		FileURL: sticker.FileURL,
		Width:   sticker.Width,
		Height:  sticker.Height,
	}
}
