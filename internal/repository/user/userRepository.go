package user

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
)

var (
	ErrUserNotFound         = errors.New("User not found")
	ErrEmailAlreadyRegister = errors.New("Email already register")
	ErrLoginAlreadyRegister = errors.New("Login already register")
)

type UserRepository struct {
	storage    map[uuid.UUID]*domain.User
	emailIndex map[string]uuid.UUID
	loginIndex map[string]uuid.UUID
	mu         sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		storage:    make(map[uuid.UUID]*domain.User),
		emailIndex: make(map[string]uuid.UUID),
		loginIndex: make(map[string]uuid.UUID),
	}
}

func (repo *UserRepository) Create(user *domain.User) error {
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

func (repo *UserRepository) GetUserByEmail(email string) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.emailIndex[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *UserRepository) GetUserByLogin(login string) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.loginIndex[login]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *UserRepository) GetUserByID(user_id uuid.UUID) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	user, ok := repo.storage[user_id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}
