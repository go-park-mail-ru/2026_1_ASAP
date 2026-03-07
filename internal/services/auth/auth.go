package auth

import (
	"errors"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	modelsUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type AuthServiceInterface interface {
	Register(request *dtoAuth.RequestRegistrate) error
	Login(request *dtoAuth.RequestLogin) error
	Logout(request *dtoAuth.RequestLogout) error
}

type AuthService struct {
	userRepository userRepository.UserRepositoryInterface
}

func NewAuthService(userRepository userRepository.UserRepositoryInterface) *AuthService {
	return &AuthService{userRepository: userRepository}
}

func (authService *AuthService) Register(request *dtoAuth.RequestRegistrate) error {
	_, err := authService.userRepository.GetUserByEmail(request.Email)
	if err == nil {
		return errors.New("User already registered")
	}

	passwordHashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		// Не так
		return errors.New("Password")
	}

	u := &modelsUser.User{
		Login:        request.Login,
		Email:        request.Email,
		PasswordHash: string(passwordHashed),
	}

	err = authService.userRepository.Create(u)
	if err != nil {
		// Подумать над ошибками, как их делать
		return err
	}
	return nil
}

func (authService *AuthService) Login(request *dtoAuth.RequestLogin) error {
	return nil
}

func (authService *AuthService) Logout(request *dtoAuth.RequestLogout) error {
	return nil
}
