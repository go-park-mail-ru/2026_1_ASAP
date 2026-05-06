package media

import (
	"context"
	"errors"
	"io"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	complaintmedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/media"
)

type MediaAdapter struct {
	client mediav1.MediaClient
}

func New(c mediav1.MediaClient) *MediaAdapter {
	return &MediaAdapter{client: c}
}

func (m *MediaAdapter) UploadComplaint(ctx context.Context, complaintID int64, input *complaintmedia.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", errors.New("nil file input")
	}
	data, err := io.ReadAll(io.LimitReader(input.Body, complaintmedia.MaxAvatarBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", complaintmedia.ErrEmptyFile
	}
	if int64(len(data)) > complaintmedia.MaxAvatarBytes {
		return "", complaintmedia.ErrFileTooLarge
	}

	resp, err := m.client.UploadComplaintAttachment(ctx, &mediav1.RequestUpdateComplaintAttachment{
		ComplaintId: complaintID,
		Attachment: &mediav1.File{
			Content: data,
			Type:    input.ContentType,
			Size:    int64(len(data)),
		},
	})
	if err != nil {
		return "", err
	}
	if url := resp.GetAttachmentUrl(); url != "" {
		return url, nil
	}
	return "", errors.New("empty attachment_url in response")
}
