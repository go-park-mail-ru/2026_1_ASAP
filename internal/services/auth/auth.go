package auth

import (
	"fmt"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/auth"
)

type UserServiceInterface interface {
	Register(user *models.RequestRegistrate) error
	Login(user *models.RequestLogin) error
	Logout(user *models.RequestLogout) error
}

type UserService struct {
}

func NewUserService() *UserService {
	return &UserService{}
}

func (userService *UserService) Register(user *models.RequestRegistrate) error {
	fmt.Printf("%+v\n", user)
	return nil
}

func (userService *UserService) Login(user *models.RequestLogin) error {
	return nil
}

func (userService *UserService) Logout(user *models.RequestLogout) error {
	return nil
}
