package media

import (
	"context"
	"fmt"
	"log"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MediaRepository struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewMediaRepository(ctx context.Context, cfg config.S3Config) (*MediaRepository, error) {
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
			log.Printf("We already own %s\n", cfg.Bucket)
		} else {
			return nil, fmt.Errorf("bucket %s: %w", cfg.Bucket, err)
		}
	} else {
		log.Printf("Successfully created %s\n", cfg.Bucket)
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

	return &MediaRepository{client: minioClient, bucket: cfg.Bucket, publicURL: cfg.PublicURL()}, nil
}

func (m MediaRepository) UploadAvatar(ctx context.Context, userId int64, input *media.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", profile.ErrEmptyAvatar
	}

	extension := getExtensionFromContentType(input.ContentType)

	objectName := fmt.Sprintf("avatar/%d%s", userId, extension)

	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName), nil
}

func (m MediaRepository) UploadChatAvatar(ctx context.Context, chatID int64, input *media.FileInput) (string, error) {
	if input == nil || input.Body == nil {
		return "", profile.ErrEmptyAvatar
	}

	extension := getExtensionFromContentType(input.ContentType)

	objectName := fmt.Sprintf("avatar/%d%s", chatID, extension)

	_, err := m.client.PutObject(ctx, m.bucket, objectName, input.Body, input.Size, minio.PutObjectOptions{
		ContentType: input.ContentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectName), nil
}

func (m *MediaRepository) Close() {

}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ".jpg"
	}
}
