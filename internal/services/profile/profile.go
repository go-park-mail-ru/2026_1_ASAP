package profile

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
)

type MediaRepositoryInterface interface {
	UploadAvatar(ctx context.Context, userId int64, input *media.FileInput) (string, error)
}

type ProfileRepositoryInterface interface {
	GetProfileById(ctx context.Context, profileId int64) (*domain.Profile, error)
	UploadBio(ctx context.Context, userId int64, bio string) (*domain.Profile, error)
	UploadAvatarUrl(ctx context.Context, userId int64, avatarUrl string) (*domain.Profile, error)
}

type ProfileService struct {
	profileRepository ProfileRepositoryInterface
	mediaRepository   MediaRepositoryInterface
}

func (p ProfileService) UpdateProfileBio(ctx context.Context, request *dto.RequestUpdateBio) (response *dto.ResponseUpdateProfile, err error) {
	if request.Bio == nil {
		return nil, domain.ErrEmptyBio
	}

	profile, err := p.profileRepository.UploadBio(ctx, request.UserID, *request.Bio)
	if err != nil {
		return nil, fmt.Errorf("upload bio: %w", err)
	}

	return &dto.ResponseUpdateProfile{
		UserId:   profile.UserId,
		Username: profile.Username,
		Avatar:   profile.Avatar,
		Bio:      profile.Bio,
		LastSeen: profile.LastSeen,
	}, nil
}

func (p ProfileService) UpdateProfileAvatar(ctx context.Context, request *dto.RequestUpdateAvatar) (response *dto.ResponseUpdateProfile, err error) {
	err = checkAvatar(request.File)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrFileTooLarge):
			return nil, domain.ErrAvatarTooLarge
		case errors.Is(err, media.ErrInvalidFileType):
			return nil, domain.ErrInvalidAvatarType
		case errors.Is(err, media.ErrEmptyFile):
			return nil, domain.ErrEmptyAvatar
		}

		return nil, fmt.Errorf("invalid avatar: %w", err)
	}

	avatarURL, err := p.mediaRepository.UploadAvatar(ctx, request.UserID, request.File)
	if err != nil {
		return nil, fmt.Errorf("upload avatar: %w", err)
	}

	profile, err := p.profileRepository.UploadAvatarUrl(ctx, request.UserID, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("upload avatar url: %w", err)
	}

	return &dto.ResponseUpdateProfile{
		UserId:   profile.UserId,
		Username: profile.Username,
		Avatar:   profile.Avatar,
		Bio:      profile.Bio,
	}, nil
}

func NewProfileService(profileRepository ProfileRepositoryInterface, mediaRepositoryInterface MediaRepositoryInterface) *ProfileService {
	return &ProfileService{profileRepository: profileRepository,
		mediaRepository: mediaRepositoryInterface}
}

func (p ProfileService) GetUserProfile(ctx context.Context, request *dto.RequestGetProfile) (response *dto.ResponseGetProfile, err error) {
	profile, err := p.profileRepository.GetProfileById(ctx, request.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &dto.ResponseGetProfile{
		UserId:   profile.UserId,
		Username: profile.Username,
		Avatar:   profile.Avatar,
		Bio:      profile.Bio,
	}, nil
}

const maxAvatarSize = 5 * 1024 * 1024

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func checkAvatar(input *media.FileInput) error {
	if input == nil || input.Body == nil {
		return media.ErrEmptyFile
	}
	if input.Size <= 0 {
		return media.ErrEmptyFile
	}
	if input.Size > maxAvatarSize {
		return media.ErrFileTooLarge
	}
	if !allowedAvatarTypes[input.ContentType] {
		return media.ErrInvalidFileType
	}
	return nil
}
