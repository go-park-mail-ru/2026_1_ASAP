package profile

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/profile"
	media2 "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	profile2 "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/sanitize"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=profile.go -destination=mock/profile_mock.go -package=mock
type MediaService interface {
	UploadAvatar(ctx context.Context, userId int64, input *media2.FileInput) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type ProfileRepositoryInterface interface {
	GetProfileById(ctx context.Context, profileId int64) (*pdomain.Profile, error)
	UploadBio(ctx context.Context, userId int64, bio string) (*pdomain.Profile, error)
	UploadAvatarUrl(ctx context.Context, userId int64, avatarUrl string) (*pdomain.Profile, error)
	UploadBirthDate(ctx context.Context, userId int64, birthDate *time.Time) (*pdomain.Profile, error)
	GetProfileIdByLogin(ctx context.Context, login string) (int64, error)
	UploadName(ctx context.Context, userID int64, firstName string, lastName *string) (*pdomain.Profile, error)
	DeleteUserAvatar(ctx context.Context, userId int64) (*pdomain.Profile, error)
}

type ProfileService struct {
	profileRepository ProfileRepositoryInterface
	mediaRepository   MediaService
}

func (p ProfileService) SearchIdByLogin(ctx context.Context, login *profile2.RequestSearchIdByLogin) (response *profile2.ResponseSearchIdByLogin, err error) {
	if login == nil {
		return nil, errors.New("search id by login: nil request")
	}

	userID, err := p.profileRepository.GetProfileIdByLogin(ctx, login.Login)
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to search profile by id: %w", err)
	}

	return &profile2.ResponseSearchIdByLogin{
		Login:  sanitize.Text(login.Login),
		UserId: userID,
	}, nil
}

func (p ProfileService) UpdateProfileBirthDate(ctx context.Context, userID int64, request *profile2.RequestUpdateBirthDate) (response *profile2.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile bio nil request")
	}
	if request.BirthDate == nil {
		return nil, pdomain.ErrInvalidBirthDate
	}

	date, err := time.Parse(time.DateOnly, *request.BirthDate)
	if err != nil {
		return nil, pdomain.ErrInvalidBirthDateFormat
	}
	if date.After(time.Now()) {
		return nil, pdomain.ErrInvalidBirthDate
	}
	if date.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return nil, pdomain.ErrInvalidBirthDate
	}

	profile, err := p.profileRepository.UploadBirthDate(ctx, userID, &date)
	if err != nil {
		if errors.Is(err, pdomain.ErrInvalidBirthDate) {
			return nil, pdomain.ErrInvalidBirthDate
		}
		return nil, fmt.Errorf("upload birth date: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}
	return &profile2.ResponseUpdateProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
		BirthDate: birthDate,
	}, nil
}

func (p ProfileService) UpdateProfileBio(ctx context.Context, userID int64, request *profile2.RequestUpdateBio) (response *profile2.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile bio nil request")
	}
	if request.Bio == nil {
		return nil, pdomain.ErrEmptyBio
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
	return &profile2.ResponseUpdateProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
		BirthDate: birthDate,
	}, nil
}

func (p ProfileService) UpdateProfileAvatar(ctx context.Context, userID int64, request *profile2.RequestUpdateAvatar) (response *profile2.ResponseUpdateProfile, err error) {
	if request == nil {
		return nil, errors.New("update profile avatar nil request")
	}
	err = checkAvatar(request.File)
	if err != nil {
		switch {
		case errors.Is(err, media2.ErrFileTooLarge):
			return nil, pdomain.ErrAvatarTooLarge
		case errors.Is(err, media2.ErrInvalidFileType):
			return nil, pdomain.ErrInvalidAvatarType
		case errors.Is(err, media2.ErrEmptyFile):
			return nil, pdomain.ErrEmptyAvatar
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

	return &profile2.ResponseUpdateProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		BirthDate: birthDate,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
	}, nil
}

func NewProfileService(profileRepository ProfileRepositoryInterface, mediaRepositoryInterface MediaService) *ProfileService {
	return &ProfileService{profileRepository: profileRepository,
		mediaRepository: mediaRepositoryInterface}
}

func (p ProfileService) GetUserProfile(ctx context.Context, userID int64) (response *profile2.ResponseGetProfile, err error) {
	profile, err := p.profileRepository.GetProfileById(ctx, userID)
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}

	return &profile2.ResponseGetProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		BirthDate: birthDate,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
	}, nil
}

func (p ProfileService) UpdateProfileName(ctx context.Context, userID int64, request *profile2.RequestUpdateName) (*profile2.ResponseUpdateProfile, error) {
	if request == nil {
		return nil, errors.New("update profile bio nil request")
	}
	if request.FirstName == "" {
		return nil, pdomain.ErrEmptyFirstName
	}

	profile, err := p.profileRepository.UploadName(ctx, userID, request.FirstName, request.LastName)
	if err != nil {
		log.Print(err)
		return nil, fmt.Errorf("update profile avatar: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}
	return &profile2.ResponseUpdateProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
		BirthDate: birthDate,
	}, nil
}

func (p ProfileService) DeleteProfileAvatar(ctx context.Context, userID int64) (response *profile2.ResponseDeleteProfile, err error) {
	err = p.mediaRepository.DeleteAvatar(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("delete avatar: %w", err)
	}

	profile, err := p.profileRepository.DeleteUserAvatar(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("delete avatar url: %w", err)
	}

	var birthDate *string
	if profile.BirthDate != nil {
		date := profile.BirthDate.UTC().Format(time.DateOnly)
		birthDate = &date
	}

	return &profile2.ResponseDeleteProfile{
		UserId:    profile.UserId,
		FirstName: sanitize.Text(profile.FirstName),
		LastName:  sanitize.TextPtr(profile.LastName),
		Avatar:    profile.Avatar,
		BirthDate: birthDate,
		Bio:       sanitize.TextPtr(profile.Bio),
		LastSeen:  profile.LastSeen,
	}, nil
}

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func checkAvatar(input *media2.FileInput) error {
	if input == nil || input.Body == nil {
		return media2.ErrEmptyFile
	}
	if input.Size <= 0 {
		return media2.ErrEmptyFile
	}
	if input.Size > media2.MaxAvatarBytes {
		return media2.ErrFileTooLarge
	}
	if !allowedAvatarTypes[input.ContentType] {
		return media2.ErrInvalidFileType
	}
	return nil
}
