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
