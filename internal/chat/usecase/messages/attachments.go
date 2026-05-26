package messages

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	chatmedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	grpcprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc/clients/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

const maxAttachmentsPerMessage = 10

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=mock/profile_contacts_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages ProfileContactsInterface

type MessageMediaRepositoryInterface interface {
	UploadMessageAttachment(
		ctx context.Context,
		userID int64,
		kind mediav1.MessageAttachmentKind,
		input *chatmedia.FileInput,
		fileName string,
	) (*chatmedia.UploadMessageAttachmentResult, error)
	GetMessageVoiceMetadata(ctx context.Context, objectKey string) (*chatmedia.VoiceMetadataResult, error)
	TranscribeVoice(ctx context.Context, objectKey string) (string, error)
}

type SubscriptionChecker interface {
	IsActive(ctx context.Context, userID int64) (bool, error)
}

type ProfileContactsInterface interface {
	HasContact(ctx context.Context, userID, contactUserID int64) (bool, error)
	GetContact(ctx context.Context, userID, contactUserID int64) (*grpcprofile.ContactSnapshot, error)
}

func chatKindToMedia(kind chatv1.MessageAttachmentKind) mediav1.MessageAttachmentKind {
	switch kind {
	case chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO:
		return mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO
	case chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE:
		return mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE
	case chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE:
		return mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE
	default:
		return mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO
	}
}

func (m MessageService) UploadMessageAttachment(
	ctx context.Context,
	userID int64,
	kind chatv1.MessageAttachmentKind,
	input *chatmedia.FileInput,
	fileName string,
) (*dto.UploadAttachmentResponse, error) {
	if m.mediaRepo == nil {
		return nil, fmt.Errorf("media repository is nil")
	}
	result, err := m.mediaRepo.UploadMessageAttachment(ctx, userID, chatKindToMedia(kind), input, fileName)
	if err != nil {
		return nil, err
	}
	proxyURL := BuildAttachmentProxyURL(m.attachmentProxyBase, result.ObjectKey)
	return &dto.UploadAttachmentResponse{
		AttachmentURL: proxyURL,
		ObjectKey:     result.ObjectKey,
		MimeType:      result.MimeType,
		FileSize:      result.FileSize,
		FileName:      result.FileName,
		DurationMs:    result.DurationMs,
		Waveform:      result.Waveform,
	}, nil
}

func (m MessageService) AuthorizeMessageAttachment(ctx context.Context, userID int64, objectKey string) error {
	if err := ValidateMessageObjectKey(objectKey); err != nil {
		return domain.ErrInvalidAttachment
	}
	_, ownerID, ok := ObjectKeyFromAttachmentURL(objectKey)
	if !ok {
		return domain.ErrInvalidAttachment
	}
	if ownerID == userID {
		return nil
	}
	proxyRef := BuildAttachmentProxyURL(m.attachmentProxyBase, objectKey)
	allowed, err := m.messageRepo.CanUserAccessAttachment(ctx, userID, objectKey, proxyRef)
	if err != nil {
		return fmt.Errorf("check attachment access: %w", err)
	}
	if !allowed {
		return domain.ErrAttachmentForbidden
	}
	return nil
}

func (m MessageService) SendMessageWithAttachments(
	ctx context.Context,
	userID int64,
	chatID int64,
	req *dto.RequestSendMessageAttachments,
) (*dto.ResponseSendMessage, error) {
	if req == nil {
		return nil, fmt.Errorf("send message attachments nil request")
	}
	if len(req.Attachments) == 0 {
		return nil, domain.ErrMessageEmpty
	}
	if len(req.Attachments) > maxAttachmentsPerMessage {
		return nil, domain.ErrTooManyAttachments
	}
	if req.Text != "" && utf8.RuneCountInString(req.Text) > maxMessageRunes {
		return nil, domain.ErrMessageTooLong
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	chat, err := m.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat by id: %w", err)
	}
	if chat.Type == domain.ChatTypeChannel && userID != chat.OwnerId {
		return nil, domain.ErrOnlyOwnerCanSendMessaage
	}

	if err = validateVoiceAttachments(req.Attachments); err != nil {
		return nil, err
	}

	domainAttachments, err := m.buildDomainAttachments(ctx, userID, req.Attachments)
	if err != nil {
		return nil, err
	}

	content := req.Text
	if content == "" {
		content = buildAttachmentPreview(domainAttachments)
	}

	message := &domain.Message{
		ChatId:   chatID,
		SenderId: userID,
		Content:  content,
	}

	createdMessage, err := m.messageRepo.CreateMessageWithAttachments(ctx, message, domainAttachments)
	if err != nil {
		return nil, fmt.Errorf("messageRepo create message with attachments: %w", err)
	}

	return messageToSendResponse(createdMessage, false), nil
}

func (m MessageService) buildDomainAttachments(
	ctx context.Context,
	userID int64,
	inputs []dto.AttachmentInput,
) ([]domain.MessageAttachment, error) {
	out := make([]domain.MessageAttachment, 0, len(inputs))
	for _, in := range inputs {
		att, err := m.buildSingleAttachment(ctx, userID, in)
		if err != nil {
			return nil, err
		}
		out = append(out, att)
	}
	return out, nil
}

func (m MessageService) buildSingleAttachment(
	ctx context.Context,
	userID int64,
	in dto.AttachmentInput,
) (domain.MessageAttachment, error) {
	switch strings.ToLower(in.Type) {
	case string(domain.AttachmentTypePhoto), string(domain.AttachmentTypeVideo), string(domain.AttachmentTypeFile):
		if in.URL == "" {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		if !IsAttachmentOwnedByUser(in.URL, userID) {
			return domain.MessageAttachment{}, domain.ErrAttachmentNotOwned
		}
		objectKey, _, ok := ObjectKeyFromAttachmentURL(in.URL)
		if !ok {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		att := domain.MessageAttachment{Type: domain.AttachmentType(strings.ToLower(in.Type))}
		url := in.URL
		if m.attachmentProxyBase != "" {
			url = BuildAttachmentProxyURL(m.attachmentProxyBase, objectKey)
		}
		att.FileURL = &url
		if in.FileName != "" {
			name := in.FileName
			att.FileName = &name
		}
		return att, nil
	case string(domain.AttachmentTypeVoice):
		if in.URL == "" {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		if !IsAttachmentOwnedByUser(in.URL, userID) {
			return domain.MessageAttachment{}, domain.ErrAttachmentNotOwned
		}
		objectKey, _, ok := ObjectKeyFromAttachmentURL(in.URL)
		if !ok {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		if m.mediaRepo == nil {
			return domain.MessageAttachment{}, fmt.Errorf("media repository is nil")
		}
		meta, err := m.mediaRepo.GetMessageVoiceMetadata(ctx, objectKey)
		if err != nil {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		url := in.URL
		if m.attachmentProxyBase != "" {
			url = BuildAttachmentProxyURL(m.attachmentProxyBase, objectKey)
		}
		mime := meta.MimeType
		size := meta.FileSize
		duration := meta.DurationMs
		att := domain.MessageAttachment{
			Type:       domain.AttachmentTypeVoice,
			FileURL:    &url,
			MimeType:   &mime,
			FileSize:   &size,
			DurationMs: &duration,
			Waveform:   meta.Waveform,
		}
		return att, nil
	case string(domain.AttachmentTypeContact):
		if in.ContactUserID <= 0 {
			return domain.MessageAttachment{}, domain.ErrInvalidAttachment
		}
		if m.profileRepo == nil {
			return domain.MessageAttachment{}, fmt.Errorf("profile repository is nil")
		}
		exists, err := m.profileRepo.HasContact(ctx, userID, in.ContactUserID)
		if err != nil {
			return domain.MessageAttachment{}, fmt.Errorf("check contact: %w", err)
		}
		if !exists {
			return domain.MessageAttachment{}, domain.ErrContactNotFound
		}
		snapshot, err := m.profileRepo.GetContact(ctx, userID, in.ContactUserID)
		if err != nil {
			return domain.MessageAttachment{}, domain.ErrContactNotFound
		}
		contactID := snapshot.ContactUserID
		firstName := snapshot.FirstName
		att := domain.MessageAttachment{
			Type:             domain.AttachmentTypeContact,
			ContactUserID:    &contactID,
			ContactFirstName: &firstName,
		}
		if snapshot.LastName != nil {
			att.ContactLastName = snapshot.LastName
		}
		if snapshot.ContactAvatarURL != nil {
			att.ContactAvatarURL = snapshot.ContactAvatarURL
		}
		return att, nil
	default:
		return domain.MessageAttachment{}, domain.ErrInvalidAttachment
	}
}

func buildAttachmentPreview(attachments []domain.MessageAttachment) string {
	labels := make([]string, 0, len(attachments))
	seen := make(map[domain.AttachmentType]bool)
	for _, a := range attachments {
		if seen[a.Type] {
			continue
		}
		seen[a.Type] = true
		switch a.Type {
		case domain.AttachmentTypePhoto:
			labels = append(labels, "[Фото]")
		case domain.AttachmentTypeVideo:
			labels = append(labels, "[Видео]")
		case domain.AttachmentTypeFile:
			labels = append(labels, "[Файл]")
		case domain.AttachmentTypeContact:
			labels = append(labels, "[Контакт]")
		case domain.AttachmentTypeVoice:
			labels = append(labels, formatVoicePreview(a.DurationMs))
		}
	}
	if len(labels) == 0 {
		return "[Вложение]"
	}
	return strings.Join(labels, " ")
}

func messageToSendResponse(msg *domain.Message, read bool) *dto.ResponseSendMessage {
	if msg == nil {
		return nil
	}
	return &dto.ResponseSendMessage{
		ID:          msg.Id,
		ChatID:      msg.ChatId,
		SenderID:    msg.SenderId,
		Text:        sanitize.Text(msg.Content),
		CreatedAt:   msg.CreatedAt,
		Edited:      msg.Edited,
		Read:        read,
		Attachments: mapAttachmentsToDTOForViewer(msg.Attachments, false),
	}
}

func mapAttachmentsToDTOForViewer(attachments []domain.MessageAttachment, subscriptionActive bool) []dto.MessageAttachmentDTO {
	if len(attachments) == 0 {
		return nil
	}
	out := make([]dto.MessageAttachmentDTO, 0, len(attachments))
	for _, a := range attachments {
		item := dto.MessageAttachmentDTO{Type: string(a.Type)}
		item.URL = a.FileURL
		item.FileName = a.FileName
		item.MimeType = a.MimeType
		item.FileSize = a.FileSize
		item.ContactUserID = a.ContactUserID
		item.ContactFirstName = a.ContactFirstName
		item.ContactLastName = a.ContactLastName
		item.ContactAvatarURL = a.ContactAvatarURL
		item.DurationMs = a.DurationMs
		if len(a.Waveform) > 0 {
			item.Waveform = a.Waveform
		}
		if subscriptionActive && a.Type == domain.AttachmentTypeVoice {
			can := true
			item.CanTranscribe = &can
			if a.Transcript != nil && *a.Transcript != "" {
				item.Transcript = a.Transcript
			}
		}
		out = append(out, item)
	}
	return out
}

func validateVoiceAttachments(inputs []dto.AttachmentInput) error {
	hasVoice := false
	for _, in := range inputs {
		if strings.ToLower(in.Type) == string(domain.AttachmentTypeVoice) {
			hasVoice = true
			break
		}
	}
	if !hasVoice {
		return nil
	}
	if len(inputs) != 1 {
		return domain.ErrInvalidAttachment
	}
	if strings.ToLower(inputs[0].Type) != string(domain.AttachmentTypeVoice) {
		return domain.ErrInvalidAttachment
	}
	return nil
}

func formatVoicePreview(durationMs *int32) string {
	if durationMs == nil || *durationMs <= 0 {
		return "[Голосовое]"
	}
	totalSec := int(*durationMs) / 1000
	minutes := totalSec / 60
	seconds := totalSec % 60
	return "[Голосовое · " + strconv.Itoa(minutes) + ":" + fmt.Sprintf("%02d", seconds) + "]"
}
