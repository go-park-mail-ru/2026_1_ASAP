package dto

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

var allowedAvatarContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func (f *FileInput) Validate() error {
	if f == nil || f.Body == nil {
		return ErrEmptyFile
	}
	return ValidateAvatarFile(f.ContentType, int(f.Size))
}

// ValidateAvatarFile checks size and allowed image MIME (same rules as media service upload).
func ValidateAvatarFile(contentType string, n int) error {
	if n <= 0 {
		return ErrEmptyFile
	}
	if n > MaxAvatarBytes {
		return ErrFileTooLarge
	}
	if !allowedAvatarContentTypes[contentType] {
		return ErrInvalidFileType
	}
	return nil
}
