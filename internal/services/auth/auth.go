package auth

import (
	"errors"

	"github.com/google/uuid"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type UserRepository interface {
	Create(*domain.User) error
	GetUserByEmail(string) (*domain.User, error)
	GetUserByLogin(string) (*domain.User, error)
	GetUserByID(uuid.UUID) (*domain.User, error)
}

type SessionService interface {
	CreateSession(userID uuid.UUID) (*dtoSession.SessionDTO, error)
	GetUserID(sessionID string) (uuid.UUID, error)
	DeleteSession(sessionID string) error
}

type AuthService struct {
	userRepository UserRepository
	SessionService SessionService
}

func NewAuthService(userRepository UserRepository, sessionService SessionService) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		SessionService: sessionService,
	}
}

func (authService *AuthService) Register(request *dtoAuth.RequestRegistrate) (*dtoSession.SessionDTO, []validation.ValidationError) {
	user := &domain.User{
		Login: request.Login,
		Email: request.Email,
	}

	passwordHash, err := hash.HashPassword(request.Password)
	if err != nil {
		return nil, []validation.ValidationError{
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
			return nil, []validation.ValidationError{
				{
					Code:    "EMAIL_ALREADY_REGISTERED",
					Message: err.Error(),
					Field:   "email",
				},
			}
		} else if errors.Is(err, userRepository.ErrLoginAlreadyRegister) {
			return nil, []validation.ValidationError{
				{
					Code:    "LOGIN_ALREADY_REGISTERED",
					Message: err.Error(),
					Field:   "login",
				},
			}
		}
	}

	sessionData, err := authService.SessionService.CreateSession(user.Id)
	if err != nil {
		return nil, []validation.ValidationError{
			{
				Code:    "SESSION_ERROR",
				Message: err.Error(),
			},
		}
	}

	return sessionData, nil
}

func (authService *AuthService) Login(request *dtoAuth.RequestLogin) (*dtoSession.SessionDTO, error) {
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
