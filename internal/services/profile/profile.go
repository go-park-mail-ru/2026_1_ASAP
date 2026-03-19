package profile

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
)

type ProfileRepositoryInterface interface {
	GetProfileById(ctx context.Context, profileId int64) (*domain.Profile, error)
}

type ProfileService struct {
	profileRepository ProfileRepositoryInterface
}

func NewProfileService(profileRepository ProfileRepositoryInterface) *ProfileService {
	return &ProfileService{profileRepository: profileRepository}
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
