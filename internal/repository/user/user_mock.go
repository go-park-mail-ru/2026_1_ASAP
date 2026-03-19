package user

import (
	"sync"

	"github.com/google/uuid"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
)

func NewMockUserRepository() *UserRepository {
	user1ID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user2ID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	user3ID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	password, _ := hash.HashPassword("passWo1r&")
	users := map[uuid.UUID]*domain.User{
		user1ID: {Id: user1ID, Login: "alice", Email: "alice@example.com", PasswordHash: password},
		user2ID: {Id: user2ID, Login: "bob", Email: "bob@example.com", PasswordHash: password},
		user3ID: {Id: user3ID, Login: "charlie", Email: "charlie@example.com", PasswordHash: password},
	}

	emailIndex := map[string]uuid.UUID{
		"alice@example.com":   user1ID,
		"bob@example.com":     user2ID,
		"charlie@example.com": user3ID,
	}

	loginIndex := map[string]uuid.UUID{
		"alice":   user1ID,
		"bob":     user2ID,
		"charlie": user3ID,
	}

	return &UserRepository{
		storage:    users,
		emailIndex: emailIndex,
		loginIndex: loginIndex,
		mu:         sync.RWMutex{},
	}
}
