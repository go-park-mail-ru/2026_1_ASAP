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
		Login:        userModel.Login,
		FirstName:    userModel.FirstName,
		LastName:     null.NullStringToPtrString(userModel.LastName),
		PasswordHash: userModel.PasswordHash,
		AvatarUrl:    null.NullStringToPtrString(userModel.AvatarUrl),
		Bio:          null.NullStringToPtrString(userModel.Bio),
		LastSeenAt:   null.NullTimeToPtrTime(userModel.LastSeenAt),
		BirthDate:    null.NullTimeToPtrTime(userModel.BirthDate),
		CreatedAt:    userModel.CreatedAt,
		UpdatedAt:    userModel.UpdatedAt,
	}
}

func toModel(user *domain.User) *UserModel {
	return &UserModel{
		Id:           user.Id,
		Email:        user.Email,
		Login:        user.Login,
		FirstName:    user.FirstName,
		LastName:     null.StringPtrToNullString(user.LastName),
		PasswordHash: user.PasswordHash,
		AvatarUrl:    null.StringPtrToNullString(user.AvatarUrl),
		Bio:          null.StringPtrToNullString(user.Bio),
		LastSeenAt:   null.TimePtrToNullTime(user.LastSeenAt),
		BirthDate:    null.TimePtrToNullTime(user.BirthDate),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func toDomainProfile(profileModel *ProfileModel) *profile.Profile {
	return &profile.Profile{
		UserId:    profileModel.UserId,
		Login:     profileModel.Login,
		FirstName: profileModel.FirstName,
		Email:     profileModel.Email,
		LastName:  null.NullStringToPtrString(profileModel.LastName),
		Avatar:    null.NullStringToPtrString(profileModel.Avatar),
		Bio:       null.NullStringToPtrString(profileModel.Bio),
		BirthDate: null.NullTimeToPtrTime(profileModel.BirthDate),
		LastSeen:  null.NullTimeToPtrTime(profileModel.LastSeen),
	}
}

func toModelProfile(profile *profile.Profile) *ProfileModel {
	return &ProfileModel{
		UserId:    profile.UserId,
		Login:     profile.Login,
		FirstName: profile.FirstName,
		Email:     profile.Email,
		LastName:  null.StringPtrToNullString(profile.LastName),
		Avatar:    null.StringPtrToNullString(profile.Avatar),
		Bio:       null.StringPtrToNullString(profile.Bio),
		BirthDate: null.TimePtrToNullTime(profile.BirthDate),
		LastSeen:  null.TimePtrToNullTime(profile.LastSeen),
	}
}
