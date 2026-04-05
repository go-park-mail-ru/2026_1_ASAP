package profile

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	UploadBirthDate(ctx context.Context, userId int64, birthDate *time.Time) (*domain.Profile, error)
	GetProfileIdByLogin(ctx context.Context, login string) (int64, error)
}

type ProfileService struct {
	profileRepository ProfileRepositoryInterface
	mediaRepository   MediaRepositoryInterface
}

func (p ProfileService) SearchIdByLogin(ctx context.Context, login *dto.RequestSearchIdByLogin) (response *dto.ResponseSearchIdByLogin, err error) {
	if login == nil {
		return nil, errors.New("update profile bio nil request")
	}

	userID, err := p.profileRepository.GetProfileIdByLogin(ctx, login.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to search profile by id: %w", err)
	}

	return &dto.ResponseSearchIdByLogin{
		Login:  login.Login,
		UserId: userID,
	}, nil
}

func (p ProfileService) UpdateProfileBirthDate(ctx context.Context, userID int64, request *dto.RequestUpdateBirthDate) (response *dto.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile bio nil request")
	}
	if request.BirthDate == nil {
		return nil, domain.ErrInvalidBirthDate
	}

	date, err := time.Parse(time.DateOnly, *request.BirthDate)
	if err != nil {
		return nil, domain.ErrInvalidBirthDateFormat
	}
	if date.After(time.Now()) {
		return nil, domain.ErrInvalidBirthDate
	}

	profile, err := p.profileRepository.UploadBirthDate(ctx, userID, &date)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidBirthDate) {
			return nil, domain.ErrInvalidBirthDate
		}
		return nil, fmt.Errorf("upload birth date: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}
	return &dto.ResponseUpdateProfile{
		UserId:    profile.UserId,
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Avatar:    profile.Avatar,
		Bio:       profile.Bio,
		LastSeen:  profile.LastSeen,
		BirthDate: birthDate,
	}, nil
}

func (p ProfileService) UpdateProfileBio(ctx context.Context, userID int64, request *dto.RequestUpdateBio) (response *dto.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile bio nil request")
	}
	if request.Bio == nil {
		return nil, domain.ErrEmptyBio
	}

	profile, err := p.profileRepository.UploadBio(ctx, userID, *request.Bio)
	if err != nil {
		return nil, fmt.Errorf("upload bio: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}
	return &dto.ResponseUpdateProfile{
		UserId:    profile.UserId,
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Avatar:    profile.Avatar,
		Bio:       profile.Bio,
		LastSeen:  profile.LastSeen,
		BirthDate: birthDate,
	}, nil
}

func (p ProfileService) UpdateProfileAvatar(ctx context.Context, userID int64, request *dto.RequestUpdateAvatar) (response *dto.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile avatar nil request")
	}
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

	avatarURL, err := p.mediaRepository.UploadAvatar(ctx, userID, request.File)
	if err != nil {
		return nil, fmt.Errorf("upload avatar: %w", err)
	}

	profile, err := p.profileRepository.UploadAvatarUrl(ctx, userID, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("upload avatar url: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}

	return &dto.ResponseUpdateProfile{
		UserId:    profile.UserId,
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Avatar:    profile.Avatar,
		BirthDate: birthDate,
		Bio:       profile.Bio,
		LastSeen:  profile.LastSeen,
	}, nil
}

func NewProfileService(profileRepository ProfileRepositoryInterface, mediaRepositoryInterface MediaRepositoryInterface) *ProfileService {
	return &ProfileService{profileRepository: profileRepository,
		mediaRepository: mediaRepositoryInterface}
}

func (p ProfileService) GetUserProfile(ctx context.Context, userID int64) (response *dto.ResponseGetProfile, err error) {
	profile, err := p.profileRepository.GetProfileById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}

	return &dto.ResponseGetProfile{
		UserId:    profile.UserId,
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Avatar:    profile.Avatar,
		BirthDate: birthDate,
		Bio:       profile.Bio,
		LastSeen:  profile.LastSeen,
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
