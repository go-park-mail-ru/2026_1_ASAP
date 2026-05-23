package media

import (
	"bytes"
	"io"
	"net/http"
)

const (
	MaxAvatarBytes       = 5 * 1024 * 1024
	MaxMessagePhotoBytes = 10 * 1024 * 1024
	MaxMessageVideoBytes = 50 * 1024 * 1024
	MaxMessageFileBytes  = 20 * 1024 * 1024
)

type UploadMessageAttachmentResult struct {
	ObjectKey string
	MimeType  string
	FileSize  int64
	FileName  *string
}

type FileInput struct {
	Body        io.Reader
	ContentType string
	Size        int64
}

func NewFileInput(data []byte, contentType string) *FileInput {
	ct := contentType
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
	}
}
