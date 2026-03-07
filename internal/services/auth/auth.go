package auth

import (
	"errors"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	modelsUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"

)

type AuthServiceInterface interface {
	Register(request *dtoAuth.RequestRegistrate) []validation.ValidationError
	Login(request *dtoAuth.RequestLogin) error
	Logout(request *dtoAuth.RequestLogout) error
}

type AuthService struct {
	userRepository userRepository.UserRepositoryInterface
}

func NewAuthService(userRepository userRepository.UserRepositoryInterface) *AuthService {
	return &AuthService{userRepository: userRepository}
}

func (authService *AuthService) Register(request *dtoAuth.RequestRegistrate) []validation.ValidationError {
	user := &modelsUser.User{
		Login: request.Login,
		Email: request.Email,
	}

	passwordHash, err := hash.HashPassword(request.Password)
	if err != nil {
		return []validation.ValidationError{
			{
				Code:    "HASH_ERROR",
				Message: "Failed to hash password",
				Field:   "password",
			},
		}
	}
	user.PasswordHash = passwordHash

	err = authService.userRepository.Create(user)
	if err != nil {
		if errors.Is(err, userRepository.ErrEmailAlreadyRegister) {
			return []validation.ValidationError{
				{
					Code:    "EMAIL_ALREADY_REGISTERED",
					Message: err.Error(),
					Field:   "email",
				},
			}
		} else if errors.Is(err, userRepository.ErrLoginAlreadyRegister) {
			return []validation.ValidationError{
				{
					Code:    "LOGIN_ALREADY_REGISTERED",
					Message: err.Error(),
					Field:   "login",
				},
			}
		}
	}

	return nil
}

func (authService *AuthService) Login(request *dtoAuth.RequestLogin) error {
	return nil
}

func (authService *AuthService) Logout(request *dtoAuth.RequestLogout) error {
	return nil
}
