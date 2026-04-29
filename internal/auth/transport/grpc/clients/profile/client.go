package profile

import (
	"context"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
)

type ProfileAdapter struct {
	profileService profilev1.ProfileClient
}

func (p ProfileAdapter) Create(ctx context.Context, profileId int64, firstName string) error {
	_, err := p.profileService.CreateProfile(ctx, &profilev1.RequestCreateProfile{
		ProfileId: profileId,
		FirstName: firstName,
	})

	return err
}

func (p ProfileAdapter) UpdateName(ctx context.Context, profileId int64, firstName, secondName string) error {
	req := &profilev1.RequestUpdateName{
		UserId:    profileId,
		FirstName: firstName,
	}
	if secondName != "" {
		req.SecondName = &secondName
	}
	_, err := p.profileService.UpdateProfileName(ctx, req)
	return err
}

func (p ProfileAdapter) UpdateAvatarFromURL(ctx context.Context, profileId int64, avatarURL string) error {
	_, err := p.profileService.UpdateProfileAvatarURL(ctx, &profilev1.RequestUpdateAvatarURL{
		UserId:    profileId,
		AvatarUrl: avatarURL,
	})
	return err
}

func NewProfileAdapter(profileService profilev1.ProfileClient) *ProfileAdapter {
	return &ProfileAdapter{profileService}
}
