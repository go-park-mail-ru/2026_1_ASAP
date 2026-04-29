package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"
	dtoVK "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/vkid"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type ProfileService interface {
	Create(ctx context.Context, profileId int64, firstName string) error
	UpdateName(ctx context.Context, profileId int64, firstName, secondName string) error
}

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=auth.go -destination=mock/auth_mock.go -package=mock
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	CreateUserByVKID(ctx context.Context, vkID int64, user *domain.User) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
	GetUserByVKID(ctx context.Context, vkid int64) (*domain.User, error)
}

type VKIDRepository interface {
}

type SessionService interface {
	CreateSession(ctx context.Context, userID int64) (*dtoSession.SessionDTO, error)
	GetUserID(ctx context.Context, sessionID string) (int64, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type AuthService struct {
	userRepository UserRepository
	SessionService SessionService
	ProfileService ProfileService
}

func (authService *AuthService) AuthWithVKID(ctx context.Context, request *dtoVK.RequestAuth) (*dtoSession.SessionDTO, error) {
	login := fmt.Sprintf("vk_%d", request.VKUserID)
	firstName := strings.TrimSpace(request.FirstName)
	if firstName == "" {
		firstName = login
	}
	lastName := strings.TrimSpace(request.LastName)

	user, err := authService.userRepository.GetUserByVKID(ctx, request.VKUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			user := &domain.User{
				Login: login,
				Email: request.Email,
			}
			createdUser, err := authService.userRepository.CreateUserByVKID(ctx, request.VKUserID, user)
			if err != nil {
				return nil, fmt.Errorf("failed to create profile: %w", err)
			}
			if authService.ProfileService != nil {
				if err := authService.ProfileService.Create(ctx, createdUser.ID, firstName); err != nil {
					return nil, fmt.Errorf("failed to create profile: %w", err)
				}
				if lastName != "" {
					if err := authService.ProfileService.UpdateName(ctx, createdUser.ID, firstName, lastName); err != nil {
						return nil, fmt.Errorf("failed to update vk profile name: %w", err)
					}
				}
			}
			sessionData, err := authService.SessionService.CreateSession(ctx, createdUser.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create session: %w", err)
			}

			return sessionData, nil
		}

		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if authService.ProfileService != nil {
		_ = authService.ProfileService.Create(ctx, user.ID, firstName)
		if lastName != "" {
			_ = authService.ProfileService.UpdateName(ctx, user.ID, firstName, lastName)
		}
	}

	sessionData, err := authService.SessionService.CreateSession(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return sessionData, nil
}

func NewAuthService(userRepository UserRepository, sessionService SessionService, service ProfileService) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		SessionService: sessionService,
		ProfileService: service,
	}
}

func (authService *AuthService) Register(ctx context.Context, request *dtoAuth.RequestRegistrate) (int64, error) {
	if errs := validation.ValidationRequestRegistrate(request); len(errs) > 0 {
		return 0, fmt.Errorf("%w: %s", domain.ErrInvalidInput, errs[0].Message)
	}

	user := &domain.User{
		Login: request.Login,
		Email: request.Email,
	}

	passwordHash, err := hash.HashPassword(request.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = passwordHash

	createdUser, err := authService.userRepository.Create(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrLoginAlreadyExists) {
			return 0, domain.ErrLoginAlreadyExists
		} else if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return 0, domain.ErrEmailAlreadyExists
		}
		return 0, fmt.Errorf("failed to create profile: %w", err)
	}

	if authService.ProfileService != nil {
		if err := authService.ProfileService.Create(ctx, createdUser.ID, createdUser.Login); err != nil {
			return 0, fmt.Errorf("failed to create profile: %w", err)
		}
	}

	return createdUser.ID, nil
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

	sessionData, err := authService.SessionService.CreateSession(ctx, user.ID)
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

func (authService *AuthService) GetUserPublic(ctx context.Context, userID int64) (*dtoAuth.ResponseUserPublic, error) {
	u, err := authService.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &dtoAuth.ResponseUserPublic{
		Login: u.Login,
		Email: u.Email,
	}, nil
}
