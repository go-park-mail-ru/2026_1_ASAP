package auth

import (
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	modelsUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
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
	user := &modelsUser.User{
		Login: request.Login,
		Email: request.Email,
	}

	passwordHash, err := hash.HashPassword(request.Password)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash

	err = authService.userRepository.Create(user)
	if err != nil {
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
