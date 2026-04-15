package auth

import (
	"context"
	"errors"
	"fmt"
	"log"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=auth.go -destination=mock/auth_mock.go -package=mock
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
}

type SessionService interface {
	CreateSession(ctx context.Context, userID int64) (*dtoSession.SessionDTO, error)
	GetUserID(ctx context.Context, sessionID string) (int64, error)
	DeleteSession(ctx context.Context, sessionID string) error
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

func (authService *AuthService) Register(ctx context.Context, request *dtoAuth.RequestRegistrate) (*dtoSession.SessionDTO, error) {
	user := &domain.User{
		Login:     request.Login,
		Email:     request.Email,
		FirstName: request.Login,
	}

	passwordHash, err := hash.HashPassword(request.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = passwordHash

	createdUser, err := authService.userRepository.Create(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrLoginAlreadyExists) {
			return nil, domain.ErrLoginAlreadyExists
		} else if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, domain.ErrEmailAlreadyExists
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	sessionData, err := authService.SessionService.CreateSession(ctx, createdUser.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return sessionData, nil
}

func (authService *AuthService) Login(ctx context.Context, request *dtoAuth.RequestLogin) (*dtoSession.SessionDTO, error) {
	user, err := authService.userRepository.GetUserByLogin(ctx, request.Login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed login: %w", err)
	}
	if !hash.CheckPassword(user.PasswordHash, request.Password) {
		log.Println(err)
		return nil, domain.ErrInvalidCredentials
	}

	sessionData, err := authService.SessionService.CreateSession(ctx, user.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return sessionData, nil
}

func (authService *AuthService) Logout(ctx context.Context, request *dtoAuth.RequestLogout) error {
	err := authService.SessionService.DeleteSession(ctx, request.SessionID)
	if err != nil {
		if errors.Is(err, domainSession.ErrNotFound) {
			return domainSession.ErrNotFound
		}

		return fmt.Errorf("failed to logout: %w", err)
	}

	return nil
}
