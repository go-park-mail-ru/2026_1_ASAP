package dto

const (
	MaxMessagePhotoBytes = 10 * 1024 * 1024
	MaxMessageVideoBytes = 50 * 1024 * 1024
	MaxMessageFileBytes  = 20 * 1024 * 1024
)

type MessageAttachmentKind int

const (
	MessageAttachmentKindPhoto MessageAttachmentKind = iota + 1
	MessageAttachmentKindVideo
	MessageAttachmentKindFile
)

var allowedPhotoContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

var allowedVideoContentTypes = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"video/quicktime": true,
}

var allowedFileContentTypes = map[string]bool{
	"application/pdf":              true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"application/msword":           true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"text/plain": true,
}

func (f *FileInput) ValidateMessageAttachment(kind MessageAttachmentKind) error {
	if f == nil || f.Body == nil {
		return ErrEmptyFile
	}
	switch kind {
	case MessageAttachmentKindPhoto:
		return validateMessageFile(f.ContentType, int(f.Size), MaxMessagePhotoBytes, allowedPhotoContentTypes)
	case MessageAttachmentKindVideo:
		return validateMessageFile(f.ContentType, int(f.Size), MaxMessageVideoBytes, allowedVideoContentTypes)
	case MessageAttachmentKindFile:
		return validateMessageFile(f.ContentType, int(f.Size), MaxMessageFileBytes, allowedFileContentTypes)
	default:
		return ErrInvalidFileType
	}
}

func validateMessageFile(contentType string, n, maxBytes int, allowed map[string]bool) error {
	if n <= 0 {
		return ErrEmptyFile
	}
	if n > maxBytes {
		return ErrFileTooLarge
	}
	if !allowed[contentType] {
		return ErrInvalidFileType
	}
	return nil
}
