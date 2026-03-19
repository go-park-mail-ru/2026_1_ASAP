package user

import (
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
