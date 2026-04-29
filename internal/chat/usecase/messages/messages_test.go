package messages

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/messages/mock"
)

func TestPositiveMessageService_SendMessageEscapesHTML(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestSendMessage
	}

	tests := []struct {
		name     string
		prepare  func(*fields)
		args     args
		wantText string
	}{
		{
			name: "img onerror",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				f.msgRepo.EXPECT().
					CreateMessage(context.Background(), &domain.Message{
						ChatId:   10,
						SenderId: 55,
						Content:  `<img src=x onerror=alert(1)>`,
					}).
					Return(&domain.Message{
						Id:        1,
						ChatId:    10,
						SenderId:  55,
						Content:   `<img src=x onerror=alert(1)>`,
						CreatedAt: time.Unix(1700000000, 0).UTC(),
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 55,
				chatID: 10,
				req: &dto.RequestSendMessage{
					ChatID: 10,
					Text:   `<img src=x onerror=alert(1)>`,
				},
			},
			wantText: `&lt;img src=x onerror=alert(1)&gt;`,
		},
		{
			name: "ampersand",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(3), int64(2)).Return(true, nil)
				f.msgRepo.EXPECT().
					CreateMessage(context.Background(), &domain.Message{
						ChatId:   3,
						SenderId: 2,
						Content:  `a & b`,
					}).
					Return(&domain.Message{
						Id:        99,
						ChatId:    3,
						SenderId:  2,
						Content:   `a & b`,
						CreatedAt: time.Unix(1700000001, 0).UTC(),
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 2,
				chatID: 3,
				req: &dto.RequestSendMessage{
					ChatID: 3,
					Text:   `a & b`,
				},
			},
			wantText: `a &amp; b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo)
			resp, err := s.SendMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.wantText, resp.Text)
		})
	}
}
