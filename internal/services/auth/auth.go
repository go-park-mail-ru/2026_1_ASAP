package auth

import (
	"errors"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	modelsSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/session"
	modelsUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type AuthServiceInterface interface {
	Register(request *dtoAuth.RequestRegistrate) []validation.ValidationError
	Login(request *dtoAuth.RequestLogin) (*modelsSession.SessionData, error)
	Logout(request *dtoAuth.RequestLogout) error
}

type AuthService struct {
	userRepository userRepository.UserRepositoryInterface
	SessionService *session.SessionService
}

func NewAuthService(userRepository userRepository.UserRepositoryInterface, sessionService *session.SessionService) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		SessionService: sessionService,
	}
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

func (authService *AuthService) Login(request *dtoAuth.RequestLogin) (*modelsSession.SessionData, error) {
	user, err := authService.userRepository.GetUserByLogin(request.Login)
	if err != nil {
		return nil, errors.New("Invalid credentials")
	}

	if !hash.CheckPassword(user.PasswordHash, request.Password) {
		return nil, errors.New("Invalid credentials")
	}

	sessionData, err := authService.SessionService.CreateSession(user.Id)
	if err != nil {
		return nil, err
	}

	return sessionData, nil
}

func (authService *AuthService) Logout(request *dtoAuth.RequestLogout) error {
	err := authService.SessionService.DeleteSession(request.SessionID)
	if err != nil {
		return errors.New("Failed to logout")
	}

	return nil
}
