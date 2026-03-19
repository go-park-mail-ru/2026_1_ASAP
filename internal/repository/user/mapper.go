package user

import (
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
)

func toDomain(userModel *UserModel) *domain.User {
	return &domain.User{
		Id:           userModel.Id,
		Email:        userModel.Email,
		Username:     userModel.Username,
		PasswordHash: userModel.PasswordHash,
		AvatarUrl:    null.NullStringToPtrString(userModel.AvatarUrl),
		Bio:          null.NullStringToPtrString(userModel.Bio),
		LastSeenAt:   null.NullTimeToPtrTime(userModel.LastSeenAt),
		CreatedAt:    userModel.CreatedAt,
		UpdatedAt:    userModel.UpdatedAt,
	}
}

func toModel(user *domain.User) *UserModel {
	return &UserModel{
		Id:           user.Id,
		Email:        user.Email,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		AvatarUrl:    null.StringPtrToNullString(user.AvatarUrl),
		Bio:          null.StringPtrToNullString(user.Bio),
		LastSeenAt:   null.TimePtrToNullTime(user.LastSeenAt),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func toDomainProfile(profileModel *ProfileModel) *profile.Profile {
	return &profile.Profile{
		UserId:   profileModel.UserId,
		Username: profileModel.Username,
		Avatar:   null.NullStringToPtrString(profileModel.Avatar),
		Bio:      null.NullStringToPtrString(profileModel.Bio),
		LastSeen: null.NullTimeToPtrTime(profileModel.LastSeen),
	}
}

func toModelProfile(profile *profile.Profile) *ProfileModel {
	return &ProfileModel{
		UserId:   profile.UserId,
		Username: profile.Username,
		Avatar:   null.StringPtrToNullString(profile.Avatar),
		Bio:      null.StringPtrToNullString(profile.Bio),
		LastSeen: null.TimePtrToNullTime(profile.LastSeen),
	}
}
