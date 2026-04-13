package media

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
)

const MaxAvatarBytes = 5 * 1024 * 1024

type FileInput struct {
	Body        io.Reader
	ContentType string
	Size        int64
}

func FileInputFromMultipart(file multipart.File, header *multipart.FileHeader) (*FileInput, error) {
	if file == nil || header == nil {
		return nil, ErrEmptyFile
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, MaxAvatarBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrEmptyFile
	}
	if len(data) > MaxAvatarBytes {
		return nil, ErrFileTooLarge
	}

	ct := header.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		n := 512
		if len(data) < n {
			n = len(data)
		}
		if n > 0 {
			ct = http.DetectContentType(data[:n])
		}
	}

	return &FileInput{
		Body:        bytes.NewReader(data),
		ContentType: ct,
		Size:        int64(len(data)),
	}, nil
}
