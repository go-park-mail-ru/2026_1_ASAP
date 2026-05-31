package stickers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
)

type fakeStickerRepo struct {
	packs []domain.StickerPack
	err   error
}

func (r fakeStickerRepo) GetStickerPacks(ctx context.Context) ([]domain.StickerPack, error) {
	return r.packs, r.err
}

func TestService_GetStickerPacks(t *testing.T) {
	t.Parallel()

	title := "Funny"
	slug := "funny"
	emoji := "🙂"
	width := 128

	tests := []struct {
		name       string
		repo       Repository
		wantErr    bool
		wantTitle  string
		wantSticks int
	}{
		{
			name:    "nil repository",
			wantErr: true,
		},
		{
			name:    "repo error",
			repo:    fakeStickerRepo{err: errors.New("db down")},
			wantErr: true,
		},
		{
			name: "uses explicit title",
			repo: fakeStickerRepo{packs: []domain.StickerPack{
				{
					Id:    1,
					Name:  "funny",
					Slug:  &slug,
					Title: &title,
					Stickers: []domain.Sticker{
						{Id: 10, PackID: 1, Emoji: &emoji, Width: &width, FileURL: "https://cdn/sticker.webp"},
					},
				},
			}},
			wantTitle:  "Funny",
			wantSticks: 1,
		},
		{
			name: "falls back to name",
			repo: fakeStickerRepo{packs: []domain.StickerPack{
				{Id: 2, Name: "animals", Title: stringPtr("")},
			}},
			wantTitle: "animals",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewService(tt.repo).GetStickerPacks(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Len(t, got.Packs, 1)
			require.Equal(t, tt.wantTitle, got.Packs[0].Title)
			require.Len(t, got.Packs[0].Stickers, tt.wantSticks)
			if tt.wantSticks > 0 {
				require.Equal(t, emoji, *got.Packs[0].Stickers[0].Emoji)
				require.Equal(t, width, *got.Packs[0].Stickers[0].Width)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
