package auth

import (
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/models"
)

type UserServiceInterface interface {
	Register(user *models.User) error
	Login(user *models.User) error
	Logout(user *models.User) error
}

type UserService struct {
}

func NewUserService() *UserService {
	return &UserService{}
}

func (userService *UserService) Register(user *models.User) error {
	return nil
}

func (userService *UserService) Login(user *models.User) error {
	return nil
}

func (userService *UserService) Logout(user *models.User) error {
	return nil
}
