package messages

import (
	"context"
	"errors"
	"fmt"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func (m MessageService) isSubscriptionActive(ctx context.Context, userID int64) (bool, error) {
	if m.subscription == nil {
		return false, nil
	}
	return m.subscription.IsActive(ctx, userID)
}

func (m MessageService) TranscribeVoiceMessage(
	ctx context.Context,
	userID, chatID, messageID int64,
) (*dto.ResponseVoiceTranscript, error) {
	active, err := m.isSubscriptionActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check subscription: %w", err)
	}
	if !active {
		return nil, domain.ErrSubscriptionRequired
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	msg, err := m.messageRepo.GetMessageByID(ctx, chatID, messageID)
	if err != nil {
		return nil, err
	}

	attachmentsByMessage, err := m.messageRepo.GetAttachmentsByMessageIDs(ctx, []int64{msg.Id})
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}
	attachments := attachmentsByMessage[msg.Id]
	voiceAtt, ok := voiceAttachment(attachments)
	if !ok {
		return nil, domain.ErrNotVoiceMessage
	}

	if voiceAtt.Transcript != nil && *voiceAtt.Transcript != "" {
		return &dto.ResponseVoiceTranscript{
			ChatID:       chatID,
			MessageID:    messageID,
			AttachmentID: voiceAtt.Id,
			Transcript:   formatTextForViewer(*voiceAtt.Transcript, true),
		}, nil
	}

	if m.mediaRepo == nil {
		return nil, domain.ErrTranscriptionFailed
	}
	if voiceAtt.FileURL == nil {
		return nil, domain.ErrInvalidAttachment
	}
	objectKey, _, ok := ObjectKeyFromAttachmentURL(*voiceAtt.FileURL)
	if !ok {
		return nil, domain.ErrInvalidAttachment
	}
	if err = ValidateMessageObjectKey(objectKey); err != nil {
		return nil, domain.ErrInvalidAttachment
	}

	text, err := m.mediaRepo.TranscribeVoice(ctx, objectKey)
	if err != nil {
		if mapped := mapTranscriptionError(err); mapped != nil {
			return nil, mapped
		}
		return nil, fmt.Errorf("transcribe voice: %w", err)
	}

	updated, err := m.messageRepo.UpdateAttachmentTranscript(ctx, voiceAtt.Id, text)
	if err != nil {
		return nil, fmt.Errorf("save transcript: %w", err)
	}

	return &dto.ResponseVoiceTranscript{
		ChatID:       chatID,
		MessageID:    messageID,
		AttachmentID: updated.Id,
		Transcript:   formatTextForViewer(text, true),
	}, nil
}

func voiceAttachment(attachments []domain.MessageAttachment) (domain.MessageAttachment, bool) {
	if len(attachments) != 1 {
		return domain.MessageAttachment{}, false
	}
	if attachments[0].Type != domain.AttachmentTypeVoice {
		return domain.MessageAttachment{}, false
	}
	return attachments[0], true
}

func mapTranscriptionError(err error) error {
	if err == nil {
		return nil
	}
	_, code, _ := grpcerr.Error(err)
	if code == int32(mediav1.MediaErrorCode_MEDIA_ERROR_TRANSCRIPTION_FAILED) {
		return domain.ErrTranscriptionFailed
	}
	if errors.Is(err, domain.ErrTranscriptionFailed) {
		return domain.ErrTranscriptionFailed
	}
	return nil
}
