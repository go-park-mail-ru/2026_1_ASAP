// Package media — gRPC-клиент внешнего Media microservice; реализует usecase.MediaService.
package media

import (
	"context"
	"errors"
	"io"
	"net/http"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
)

type MediaAdapter struct {
	client mediav1.MediaClient
}

func New(c mediav1.MediaClient) *MediaAdapter {
	return &MediaAdapter{client: c}
}

func (m *MediaAdapter) UploadAvatar(ctx context.Context, userId int64, input *mediadto.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", errors.New("nil file input")
	}
	data, err := io.ReadAll(io.LimitReader(input.Body, mediadto.MaxAvatarBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("empty avatar body")
	}
	if int64(len(data)) > mediadto.MaxAvatarBytes {
		return "", errors.New("file is too large")
	}

	ct := input.ContentType
	if ct == "" || ct == "application/octet-stream" {
		n := 512
		if len(data) < n {
			n = len(data)
		}
		if n > 0 {
			ct = http.DetectContentType(data[:n])
		}
	}

	resp, err := m.client.UpdateUserAvatar(ctx, &mediav1.RequestUpdateUserAvatar{
		UserId: userId,
		Avatar: &mediav1.File{
			Content: data,
			Type:    ct,
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

func (m *MediaAdapter) DeleteAvatar(ctx context.Context, userID int64) error {
	_, err := m.client.DeleteUserAvatar(ctx, &mediav1.RequestDeleteUserAvatar{UserId: userID})
	return err
}
