package media

import (
	"context"
	"errors"
	"io"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	chatmedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
)

type MediaAdapter struct {
	client mediav1.MediaClient
}

func New(c mediav1.MediaClient) *MediaAdapter {
	return &MediaAdapter{client: c}
}

func (m *MediaAdapter) UploadChatAvatar(ctx context.Context, chatID int64, input *chatmedia.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", errors.New("nil file input")
	}
	data, err := io.ReadAll(io.LimitReader(input.Body, chatmedia.MaxAvatarBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", chatmedia.ErrEmptyFile
	}
	if int64(len(data)) > chatmedia.MaxAvatarBytes {
		return "", chatmedia.ErrFileTooLarge
	}

	resp, err := m.client.UploadChatAvatar(ctx, &mediav1.RequestUpdateChatAvatar{
		ChatId: chatID,
		Avatar: &mediav1.File{
			Content: data,
			Type:    input.ContentType,
			Size:    int64(len(data)),
		},
	})
	if err != nil {
		return "", err
	}
	if url := resp.GetAvatarUrl(); url != "" {
		return url, nil
	}
	return "", errors.New("empty avatar_url in response")
}

func (m *MediaAdapter) UploadMessageAttachment(
	ctx context.Context,
	userID int64,
	kind mediav1.MessageAttachmentKind,
	input *chatmedia.FileInput,
	fileName string,
) (*chatmedia.UploadMessageAttachmentResult, error) {
	if input == nil || input.Body == nil {
		return nil, errors.New("nil file input")
	}
	maxBytes := int64(chatmedia.MaxMessagePhotoBytes)
	switch kind {
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO:
		maxBytes = int64(chatmedia.MaxMessageVideoBytes)
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE:
		maxBytes = int64(chatmedia.MaxMessageFileBytes)
	}
	data, err := io.ReadAll(io.LimitReader(input.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, chatmedia.ErrEmptyFile
	}
	if int64(len(data)) > maxBytes {
		return nil, chatmedia.ErrFileTooLarge
	}

	req := &mediav1.RequestUploadMessageAttachment{
		UserId: userID,
		Kind:   kind,
		File: &mediav1.File{
			Content: data,
			Type:    input.ContentType,
			Size:    int64(len(data)),
		},
	}
	if fileName != "" {
		req.FileName = &fileName
	}
	resp, err := m.client.UploadMessageAttachment(ctx, req)
	if err != nil {
		return nil, err
	}
	if key := resp.GetObjectKey(); key != "" {
		result := &chatmedia.UploadMessageAttachmentResult{
			ObjectKey: key,
			MimeType:  resp.GetMimeType(),
			FileSize:  resp.GetFileSize(),
		}
		if name := resp.GetFileName(); name != "" {
			result.FileName = &name
		} else if fileName != "" {
			result.FileName = &fileName
		}
		return result, nil
	}
	return nil, errors.New("empty object_key in response")
}
