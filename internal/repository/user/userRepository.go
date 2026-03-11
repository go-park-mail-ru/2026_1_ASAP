package user

import (
	"errors"
	"sync"
	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound         = errors.New("User not found")
	ErrEmailAlreadyRegister = errors.New("Email already register")
	ErrLoginAlreadyRegister = errors.New("Login already register")
)

type UserRepositoryInterface interface {
	Create(*models.User) error
	GetUserByEmail(string) (*models.User, error)
	GetUserByLogin(string) (*models.User, error)
	GetUserByID(uuid.UUID) (*models.User, error)
}

type UserRepository struct {
	storage    map[uuid.UUID]*models.User
	emailIndex map[string]uuid.UUID
	loginIndex map[string]uuid.UUID
	mu         sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		storage:    make(map[uuid.UUID]*models.User),
		emailIndex: make(map[string]uuid.UUID),
		loginIndex: make(map[string]uuid.UUID),
	}
}

func (repo *UserRepository) Create(user *models.User) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if _, exists := repo.emailIndex[user.Email]; exists {
		return ErrEmailAlreadyRegister
	}
	if _, exists := repo.loginIndex[user.Login]; exists {
		return ErrLoginAlreadyRegister
	}

	user.Id = uuid.New()
	repo.storage[user.Id] = user
	repo.emailIndex[user.Email] = user.Id
	repo.loginIndex[user.Login] = user.Id
	return nil
}

func (repo *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.emailIndex[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *UserRepository) GetUserByLogin(login string) (*models.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.loginIndex[login]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *UserRepository) GetUserByID(user_id uuid.UUID) (*models.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	user, ok := repo.storage[user_id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}
