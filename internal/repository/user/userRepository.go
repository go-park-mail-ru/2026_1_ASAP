package user

import (
	"errors"
	"log"
	"sync"

	models "github.com/go-park-mail-ru/2026_1_ASAP/internal/models/user"
)

var (
	ErrUserAlreadyRegister = errors.New("User already register")
	ErrUserNotFound        = errors.New("User not found")
)

type UserRepositoryInterface interface {
	Create(user *models.User) error
	Update(user *models.User) error
	GetUserByEmail(string) (*models.User, error)
}

type UserRepository struct {
	storage map[string]*models.User
	mu      sync.RWMutex
	nextId  int
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		storage: make(map[string]*models.User),
		nextId:  1,
	}
}

func (userRepository *UserRepository) Create(user *models.User) error {
	userRepository.mu.Lock()
	defer userRepository.mu.Unlock()

	if _, ok := userRepository.storage[user.Email]; ok {
		return ErrUserAlreadyRegister
	}
	user.Id = userRepository.nextId
	userRepository.nextId++
	userRepository.storage[user.Email] = user
	log.Println("%+v", userRepository.storage)
	return nil
}

func (userRepository *UserRepository) Update(user *models.User) error {
	return nil
}

func (userRepository *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	userRepository.mu.RLock()
	defer userRepository.mu.RUnlock()

	if value, ok := userRepository.storage[email]; ok {
		return value, nil
	} else {
		return nil, ErrUserNotFound
	}

}
