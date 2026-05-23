package repository

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/s3log"
	"github.com/google/uuid"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

type MediaRepository struct {
	client    *minio.Client
	logger    *zap.Logger
	bucket    string
	publicURL string
}

func NewMediaRepository(ctx context.Context, cfg config.S3Config, logger *zap.Logger) (*MediaRepository, error) {
	minioClient, err := minio.New(cfg.Endpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	err = minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, cfg.Bucket)
		if errBucketExists == nil && exists {
			if logger != nil {
				logger.Info("s3 bucket already exists", zap.String("bucket", cfg.Bucket))
			}
		} else {
			return nil, fmt.Errorf("bucket %s: %w", cfg.Bucket, err)
		}
	} else {
		if logger != nil {
			logger.Info("s3 bucket created", zap.String("bucket", cfg.Bucket))
		}
	}
	policy := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": ["s3:GetObject"],
            "Effect": "Allow",
            "Principal": {"AWS": ["*"]},
            "Resource": ["arn:aws:s3:::` + cfg.Bucket + `/*"],
            "Sid": ""
        }
    ]
	}`
	err = minioClient.SetBucketPolicy(ctx, cfg.Bucket, policy)
	if err != nil {
		return nil, fmt.Errorf("set bucket policy: %w", err)
	}

	return &MediaRepository{client: minioClient, bucket: cfg.Bucket, publicURL: cfg.PublicURL(), logger: logger}, nil
}

func (m *MediaRepository) UploadAvatar(ctx context.Context, userId int64, input *mediadto.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", mediadto.ErrEmptyFile
	}

	extension := getExtensionFromContentType(input.ContentType)

	objectName := fmt.Sprintf("avatar/user_%d_%d%s", userId, time.Now().UnixNano(), extension)

	start := time.Now()
	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	s3log.LogOp(ctx, m.log(ctx), "UploadAvatar", objectName, start, err, []any{userId, input.ContentType, input.Size})
	if err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName), nil
}

func (m *MediaRepository) UploadChatAvatar(ctx context.Context, chatID int64, input *mediadto.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", mediadto.ErrEmptyFile
	}

	extension := getExtensionFromContentType(input.ContentType)

	objectName := fmt.Sprintf("avatar/chat_%d_%d%s", chatID, time.Now().UnixNano(), extension)

	start := time.Now()
	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	s3log.LogOp(ctx, m.log(ctx), "UploadChatAvatar", objectName, start, err, []any{chatID, input.ContentType, input.Size})
	if err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName), nil
}

func (m *MediaRepository) UploadComplaint(ctx context.Context, complaintID int64, input *mediadto.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", mediadto.ErrEmptyFile
	}

	extension := getExtensionFromContentType(input.ContentType)
	objectName := fmt.Sprintf("complaint/%d_%d%s", complaintID, time.Now().UnixNano(), extension)

	start := time.Now()
	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	s3log.LogOp(ctx, m.log(ctx), "UploadComplaint", objectName, start, err, []any{complaintID, input.ContentType, input.Size})
	if err != nil {
		return "", fmt.Errorf("upload complaint attachment: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName), nil
}

type MessageAttachmentObject struct {
	ObjectKey   string
	ContentType string
	Size        int64
}

func (m *MediaRepository) UploadMessageAttachment(
	ctx context.Context,
	userID int64,
	kind mediadto.MessageAttachmentKind,
	input *mediadto.FileInput,
) (*MessageAttachmentObject, error) {
	if input == nil || input.Body == nil {
		return nil, mediadto.ErrEmptyFile
	}
	if err := input.ValidateMessageAttachment(kind); err != nil {
		return nil, err
	}

	extension := getExtensionFromContentType(input.ContentType)
	objectName := fmt.Sprintf("message/%d/%s_%d%s", userID, uuid.NewString(), time.Now().UnixNano(), extension)

	start := time.Now()
	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	s3log.LogOp(ctx, m.log(ctx), "UploadMessageAttachment", objectName, start, err, []any{userID, kind, input.ContentType, input.Size})
	if err != nil {
		return nil, fmt.Errorf("upload message attachment: %w", err)
	}

	return &MessageAttachmentObject{
		ObjectKey:   objectName,
		ContentType: input.ContentType,
		Size:        input.Size,
	}, nil
}

func (m *MediaRepository) GetMessageAttachment(ctx context.Context, objectKey string) ([]byte, string, error) {
	if err := validateMessageObjectKey(objectKey); err != nil {
		return nil, "", err
	}
	start := time.Now()
	obj, err := m.client.GetObject(ctx, m.bucket, objectKey, minio.GetObjectOptions{})
	s3log.LogOp(ctx, m.log(ctx), "GetMessageAttachment", objectKey, start, err, []any{objectKey})
	if err != nil {
		return nil, "", fmt.Errorf("get message attachment: %w", err)
	}
	defer func() { _ = obj.Close() }()

	info, err := obj.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("stat message attachment: %w", err)
	}
	ct := info.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}

	const maxGet = mediadto.MaxMessageVideoBytes
	data, err := io.ReadAll(io.LimitReader(obj, maxGet+1))
	if err != nil {
		return nil, "", fmt.Errorf("read message attachment: %w", err)
	}
	if len(data) > maxGet {
		return nil, "", mediadto.ErrFileTooLarge
	}
	return data, ct, nil
}

func validateMessageObjectKey(objectKey string) error {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if !strings.HasPrefix(key, "message/") {
		return fmt.Errorf("invalid object key")
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[2] == "" {
		return fmt.Errorf("invalid object key format")
	}
	return nil
}

func (m *MediaRepository) DeleteAvatar(ctx context.Context, userID int64) error {
	extensions := []string{".jpg", ".png", ".webp"}

	for _, ext := range extensions {
		objectName := fmt.Sprintf("avatar/%d%s", userID, ext)

		start := time.Now()
		err := m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
		s3log.LogOp(ctx, m.log(ctx), "DeleteAvatar", objectName, start, err, []any{userID, ext})
		if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
			return fmt.Errorf("failed to delete avatar: %w", err)
		}
	}
	return nil
}

func (m *MediaRepository) Close() {}

func (m *MediaRepository) log(ctx context.Context) *zap.Logger {
	base := m.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	case "application/pdf":
		return ".pdf"
	case "application/zip", "application/x-zip-compressed":
		return ".zip"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "text/plain":
		return ".txt"
	default:
		return ".bin"
	}
}
