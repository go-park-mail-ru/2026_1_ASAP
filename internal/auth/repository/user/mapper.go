package user

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
)

func toDomain(userModel *UserModel) *domain.User {
	return &domain.User{
		ID:           userModel.ID,
		Email:        userModel.Email,
		Login:        userModel.Login,
		PasswordHash: userModel.PasswordHash,
		VKID:         userModel.VKID,
		CreatedAt:    userModel.CreatedAt,
		UpdatedAt:    userModel.UpdatedAt,
	}
}

func toModel(user *domain.User) *UserModel {
	return &UserModel{
		ID:           user.ID,
		Email:        user.Email,
		Login:        user.Login,
		PasswordHash: user.PasswordHash,
		VKID:         user.VKID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}
