package messages

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages/mock"
)

type subscriptionStub struct {
	active bool
	err    error
}

func (s subscriptionStub) IsActive(context.Context, int64) (bool, error) {
	return s.active, s.err
}

func TestTranscribeVoiceMessage_SubscriptionRequired(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)

	s := NewMessageService(msgRepo, chatRepo, nil, nil, "http://localhost:8088", subscriptionStub{active: false})

	_, err := s.TranscribeVoiceMessage(context.Background(), 1, 10, 100)
	require.ErrorIs(t, err, domain.ErrSubscriptionRequired)
}

func TestTranscribeVoiceMessage_CachedTranscript(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	msgRepo := mock.NewMockMessageRepositoryInterface(ctrl)
	chatRepo := mock.NewMockChatRepositoryInterface(ctrl)

	transcript := "hello"
	fileURL := "http://localhost:8088/api/v1/messages/attachments/message/5/voice.ogg"

	chatRepo.EXPECT().IsMember(gomock.Any(), int64(10), int64(1)).Return(true, nil)
	msgRepo.EXPECT().GetMessageByID(gomock.Any(), int64(10), int64(100)).Return(&domain.Message{
		Id:     100,
		ChatId: 10,
	}, nil)
	msgRepo.EXPECT().GetAttachmentsByMessageIDs(gomock.Any(), []int64{100}).Return(map[int64][]domain.MessageAttachment{
		100: {{
			Id:         7,
			MessageId:  100,
			Type:       domain.AttachmentTypeVoice,
			FileURL:    &fileURL,
			Transcript: &transcript,
		}},
	}, nil)

	s := NewMessageService(msgRepo, chatRepo, nil, nil, "http://localhost:8088", subscriptionStub{active: true})

	resp, err := s.TranscribeVoiceMessage(context.Background(), 1, 10, 100)
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Transcript)
	require.Equal(t, int64(7), resp.AttachmentID)
}

func TestMapAttachmentsToDTOForViewer_Subscription(t *testing.T) {
	t.Parallel()
	transcript := "text"
	attachments := []domain.MessageAttachment{{
		Type:       domain.AttachmentTypeVoice,
		Transcript: &transcript,
	}}

	withSub := mapAttachmentsToDTOForViewer(attachments, true)
	require.NotNil(t, withSub[0].CanTranscribe)
	require.True(t, *withSub[0].CanTranscribe)
	require.Equal(t, &transcript, withSub[0].Transcript)

	withoutSub := mapAttachmentsToDTOForViewer(attachments, false)
	require.Nil(t, withoutSub[0].CanTranscribe)
	require.Nil(t, withoutSub[0].Transcript)
}

func TestMapAttachmentsToDTOForViewer_ContentFilter(t *testing.T) {
	t.Parallel()
	attachments := []domain.MessageAttachment{
		{Type: domain.AttachmentTypePhoto, IsCapybara: true},
		{Type: domain.AttachmentTypePhoto, IsCapybara: false},
	}

	withSub := mapAttachmentsToDTOForViewer(attachments, true)
	require.True(t, withSub[0].IsBlur)
	require.False(t, withSub[1].IsBlur)

	withoutSub := mapAttachmentsToDTOForViewer(attachments, false)
	require.False(t, withoutSub[0].IsBlur)
	require.False(t, withoutSub[1].IsBlur)
}

func TestFormatTextForViewer_Profanity(t *testing.T) {
	t.Parallel()
	raw := "блять"
	require.Equal(t, "блять", formatTextForViewer(raw, false))
	require.Equal(t, "***", formatTextForViewer(raw, true))
}

func TestPresentSendMessageForViewer(t *testing.T) {
	t.Parallel()
	resp := &dto.ResponseSendMessage{
		ContentRaw: "блять",
		Text:       "блять",
		Attachments: []dto.MessageAttachmentDTO{
			{Type: "photo", IsCapybara: true},
		},
	}
	out := PresentSendMessageForViewer(resp, true)
	require.Equal(t, "***", out.Text)
	require.True(t, out.Attachments[0].IsBlur)
}
