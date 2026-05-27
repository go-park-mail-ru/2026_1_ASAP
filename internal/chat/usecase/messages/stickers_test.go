package messages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages/mock"
)

func TestMessageService_SendSticker(t *testing.T) {
	t.Parallel()

	stickerID := int64(77)
	width := 512
	emoji := "🔥"
	now := time.Unix(1700000000, 0).UTC()

	tests := []struct {
		name        string
		req         *dto.RequestSendSticker
		prepare     func(*mock.MockMessageRepositoryInterface, *mock.MockChatRepositoryInterface, *mock.MockStickerRepositoryInterface)
		wantErr     error
		wantAnyErr  bool
		wantSticker bool
	}{
		{
			name:       "nil request",
			req:        nil,
			wantAnyErr: true,
		},
		{
			name:    "invalid sticker",
			req:     &dto.RequestSendSticker{StickerID: 0},
			wantErr: domain.ErrInvalidSticker,
		},
		{
			name:       "nil sticker repository",
			req:        &dto.RequestSendSticker{StickerID: stickerID},
			prepare:    nil,
			wantAnyErr: true,
		},
		{
			name: "user not member",
			req:  &dto.RequestSendSticker{StickerID: stickerID},
			prepare: func(_ *mock.MockMessageRepositoryInterface, chatRepo *mock.MockChatRepositoryInterface, _ *mock.MockStickerRepositoryInterface) {
				chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(false, nil)
			},
			wantErr: domain.ErrMessageNotMember,
		},
		{
			name: "channel not owner",
			req:  &dto.RequestSendSticker{StickerID: stickerID},
			prepare: func(_ *mock.MockMessageRepositoryInterface, chatRepo *mock.MockChatRepositoryInterface, _ *mock.MockStickerRepositoryInterface) {
				chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				chatRepo.EXPECT().GetChatByID(context.Background(), int64(10)).Return(&domain.Chat{
					Id:      10,
					Type:    domain.ChatTypeChannel,
					OwnerId: 99,
				}, nil)
			},
			wantErr: domain.ErrOnlyOwnerCanSendMessaage,
		},
		{
			name: "sticker not found",
			req:  &dto.RequestSendSticker{StickerID: stickerID},
			prepare: func(_ *mock.MockMessageRepositoryInterface, chatRepo *mock.MockChatRepositoryInterface, stickerRepo *mock.MockStickerRepositoryInterface) {
				chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				chatRepo.EXPECT().GetChatByID(context.Background(), int64(10)).Return(&domain.Chat{Id: 10, Type: domain.ChatTypeGroup}, nil)
				stickerRepo.EXPECT().GetStickerByID(context.Background(), stickerID).Return(nil, domain.ErrStickerNotFound)
			},
			wantErr: domain.ErrStickerNotFound,
		},
		{
			name: "create error",
			req:  &dto.RequestSendSticker{StickerID: stickerID},
			prepare: func(msgRepo *mock.MockMessageRepositoryInterface, chatRepo *mock.MockChatRepositoryInterface, stickerRepo *mock.MockStickerRepositoryInterface) {
				chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				chatRepo.EXPECT().GetChatByID(context.Background(), int64(10)).Return(&domain.Chat{Id: 10, Type: domain.ChatTypeGroup}, nil)
				stickerRepo.EXPECT().GetStickerByID(context.Background(), stickerID).Return(&domain.Sticker{Id: stickerID}, nil)
				msgRepo.EXPECT().CreateMessage(context.Background(), &domain.Message{
					ChatId:    10,
					SenderId:  55,
					Content:   stickerMessagePreview,
					StickerId: &stickerID,
				}).Return(nil, errors.New("db down"))
			},
			wantAnyErr: true,
		},
		{
			name: "success",
			req:  &dto.RequestSendSticker{StickerID: stickerID},
			prepare: func(msgRepo *mock.MockMessageRepositoryInterface, chatRepo *mock.MockChatRepositoryInterface, stickerRepo *mock.MockStickerRepositoryInterface) {
				chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				chatRepo.EXPECT().GetChatByID(context.Background(), int64(10)).Return(&domain.Chat{Id: 10, Type: domain.ChatTypeGroup}, nil)
				stickerRepo.EXPECT().GetStickerByID(context.Background(), stickerID).Return(&domain.Sticker{
					Id:      stickerID,
					PackID:  3,
					FileURL: "https://cdn/sticker.webp",
					Emoji:   &emoji,
					Width:   &width,
				}, nil)
				msgRepo.EXPECT().CreateMessage(context.Background(), &domain.Message{
					ChatId:    10,
					SenderId:  55,
					Content:   stickerMessagePreview,
					StickerId: &stickerID,
				}).Return(&domain.Message{
					Id:        101,
					ChatId:    10,
					SenderId:  55,
					Content:   stickerMessagePreview,
					StickerId: &stickerID,
					CreatedAt: now,
				}, nil)
			},
			wantSticker: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
			chatRepo := mock.NewMockChatRepositoryInterface(ctrl)
			var stickerRepo *mock.MockStickerRepositoryInterface
			var stickerArg StickerRepositoryInterface
			if tt.name != "nil sticker repository" {
				stickerRepo = mock.NewMockStickerRepositoryInterface(ctrl)
				stickerArg = stickerRepo
			}
			if tt.prepare != nil {
				tt.prepare(msgRepo, chatRepo, stickerRepo)
			}

			svc := NewMessageService(msgRepo, chatRepo, nil, nil, "http://localhost", nil, stickerArg)
			got, err := svc.SendSticker(context.Background(), 55, 10, tt.req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantAnyErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, int64(101), got.ID)
			require.NotNil(t, got.Sticker)
			require.Equal(t, tt.wantSticker, got.Sticker != nil)
			require.Equal(t, stickerID, got.Sticker.ID)
			require.Equal(t, emoji, *got.Sticker.Emoji)
			require.Equal(t, width, *got.Sticker.Width)
		})
	}
}

func TestMessageService_StickersByMessage(t *testing.T) {
	t.Parallel()

	id1 := int64(1)
	id2 := int64(2)

	tests := []struct {
		name    string
		repo    bool
		input   []*domain.Message
		prepare func(*mock.MockStickerRepositoryInterface)
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil repo",
			input:   []*domain.Message{{StickerId: &id1}},
			wantLen: 0,
		},
		{
			name:    "no sticker ids",
			repo:    true,
			input:   []*domain.Message{nil, {}, {StickerId: nil}},
			wantLen: 0,
		},
		{
			name:  "unique ids",
			repo:  true,
			input: []*domain.Message{{StickerId: &id1}, {StickerId: &id1}, {StickerId: &id2}},
			prepare: func(stickerRepo *mock.MockStickerRepositoryInterface) {
				stickerRepo.EXPECT().
					GetStickersByIDs(context.Background(), []int64{id1, id2}).
					Return(map[int64]domain.Sticker{id1: {Id: id1}, id2: {Id: id2}}, nil)
			},
			wantLen: 2,
		},
		{
			name:  "repo error",
			repo:  true,
			input: []*domain.Message{{StickerId: &id1}},
			prepare: func(stickerRepo *mock.MockStickerRepositoryInterface) {
				stickerRepo.EXPECT().GetStickersByIDs(context.Background(), []int64{id1}).Return(nil, errors.New("db down"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var stickerRepo *mock.MockStickerRepositoryInterface
			var stickerArg StickerRepositoryInterface
			if tt.repo {
				stickerRepo = mock.NewMockStickerRepositoryInterface(ctrl)
				stickerArg = stickerRepo
			}
			if tt.prepare != nil {
				tt.prepare(stickerRepo)
			}

			svc := NewMessageService(nil, nil, nil, nil, "", nil, stickerArg)
			got, err := svc.stickersByMessage(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
		})
	}
}

func TestStickerDTOFromMessage(t *testing.T) {
	t.Parallel()

	id := int64(5)
	slug := "wave"
	sticker := domain.Sticker{Id: id, PackID: 9, Slug: &slug}

	require.Nil(t, stickerDTOFromMessage(nil, map[int64]domain.Sticker{id: sticker}))
	require.Nil(t, stickerDTOFromMessage(&domain.Message{}, map[int64]domain.Sticker{id: sticker}))
	require.Nil(t, stickerDTOFromMessage(&domain.Message{StickerId: &id}, map[int64]domain.Sticker{}))

	got := stickerDTOFromMessage(&domain.Message{StickerId: &id}, map[int64]domain.Sticker{id: sticker})
	require.NotNil(t, got)
	require.Equal(t, id, got.ID)
	require.Equal(t, slug, *got.Slug)
}
