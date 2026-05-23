package messages

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	grpcprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc/clients/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages/mock"
)

func TestSendMessageWithAttachments_HappyPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)

	chatRepo.EXPECT().IsMember(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	chatRepo.EXPECT().GetChatByID(gomock.Any(), int64(1)).Return(&domain.Chat{Id: 1, Type: domain.ChatTypeGroup}, nil)
	msgRepo.EXPECT().CreateMessageWithAttachments(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, msg *domain.Message, atts []domain.MessageAttachment) (*domain.Message, error) {
			require.Equal(t, "caption", msg.Content)
			require.Len(t, atts, 1)
			require.Equal(t, domain.AttachmentTypePhoto, atts[0].Type)
			msg.Id = 99
			msg.Attachments = atts
			return msg, nil
		},
	)

	s := NewMessageService(msgRepo, chatRepo, nil, nil, "http://localhost:8088")
	resp, err := s.SendMessageWithAttachments(context.Background(), 10, 1, &dto.RequestSendMessageAttachments{
		ChatID: 1,
		Text:   "caption",
		Attachments: []dto.AttachmentInput{{
			Type: "photo",
			URL:  "http://localhost:8088/api/v1/messages/attachments/message/10/uuid_1.jpg",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, int64(99), resp.ID)
	require.Len(t, resp.Attachments, 1)
}

func TestSendMessageWithAttachments_RejectsForeignURL(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)

	chatRepo.EXPECT().IsMember(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	chatRepo.EXPECT().GetChatByID(gomock.Any(), int64(1)).Return(&domain.Chat{Id: 1, Type: domain.ChatTypeGroup}, nil)

	s := NewMessageService(msgRepo, chatRepo, nil, nil, "http://localhost:8088")
	_, err := s.SendMessageWithAttachments(context.Background(), 10, 1, &dto.RequestSendMessageAttachments{
		ChatID: 1,
		Attachments: []dto.AttachmentInput{{
			Type: "photo",
			URL:  "http://localhost:8088/api/v1/messages/attachments/message/999/uuid_1.jpg",
		}},
	})
	require.ErrorIs(t, err, domain.ErrAttachmentNotOwned)
}

func TestSendMessageWithAttachments_ContactNotInBook(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)
	profile := mock.NewMockProfileContactsInterface(ctrl)

	chatRepo.EXPECT().IsMember(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	chatRepo.EXPECT().GetChatByID(gomock.Any(), int64(1)).Return(&domain.Chat{Id: 1, Type: domain.ChatTypeGroup}, nil)
	profile.EXPECT().HasContact(gomock.Any(), int64(10), int64(42)).Return(false, nil)

	s := NewMessageService(msgRepo, chatRepo, nil, profile, "http://localhost:8088")
	_, err := s.SendMessageWithAttachments(context.Background(), 10, 1, &dto.RequestSendMessageAttachments{
		ChatID: 1,
		Attachments: []dto.AttachmentInput{{
			Type:          "contact",
			ContactUserID: 42,
		}},
	})
	require.ErrorIs(t, err, domain.ErrContactNotFound)
}

func TestSendMessageWithAttachments_ContactSnapshot(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)
	profile := mock.NewMockProfileContactsInterface(ctrl)

	chatRepo.EXPECT().IsMember(gomock.Any(), int64(1), int64(10)).Return(true, nil)
	chatRepo.EXPECT().GetChatByID(gomock.Any(), int64(1)).Return(&domain.Chat{Id: 1, Type: domain.ChatTypeGroup}, nil)
	profile.EXPECT().HasContact(gomock.Any(), int64(10), int64(42)).Return(true, nil)
	lastName := "Ivanov"
	profile.EXPECT().GetContact(gomock.Any(), int64(10), int64(42)).Return(&grpcprofile.ContactSnapshot{
		ContactUserID: 42,
		FirstName:     "Ivan",
		LastName:      &lastName,
	}, nil)
	msgRepo.EXPECT().CreateMessageWithAttachments(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, msg *domain.Message, atts []domain.MessageAttachment) (*domain.Message, error) {
			require.Equal(t, "[Контакт]", msg.Content)
			require.Equal(t, "Ivan", *atts[0].ContactFirstName)
			msg.Id = 1
			msg.Attachments = atts
			return msg, nil
		},
	)

	s := NewMessageService(msgRepo, chatRepo, nil, profile, "http://localhost:8088")
	resp, err := s.SendMessageWithAttachments(context.Background(), 10, 1, &dto.RequestSendMessageAttachments{
		ChatID: 1,
		Attachments: []dto.AttachmentInput{{
			Type:          "contact",
			ContactUserID: 42,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "contact", resp.Attachments[0].Type)
}

func TestBuildAttachmentPreview(t *testing.T) {
	t.Parallel()
	preview := buildAttachmentPreview([]domain.MessageAttachment{
		{Type: domain.AttachmentTypePhoto},
		{Type: domain.AttachmentTypeContact},
	})
	require.Equal(t, "[Фото] [Контакт]", preview)
}
