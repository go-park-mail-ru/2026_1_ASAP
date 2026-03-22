package user

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(ctx context.Context, cfg config.PostgresConfig) (*UserRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &UserRepository{db: pool}, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	userModel := toModel(user)
	err := r.db.QueryRow(ctx,
		`INSERT INTO users
        (username, email, password_hash, avatar_url, bio, last_seen, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING id`,
		userModel.Username, userModel.Email, userModel.PasswordHash,
		userModel.AvatarUrl, userModel.Bio, userModel.LastSeenAt,
		userModel.CreatedAt, userModel.UpdatedAt,
	).Scan(&userModel.Id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return nil, domain.ErrLoginAlreadyExists
			case "users_email_key":
				return nil, domain.ErrEmailAlreadyExists
			}
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return toDomain(userModel), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar_url, bio, last_seen, created_at, updated_at
         FROM users WHERE email=$1`, email)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by email: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByLogin(ctx context.Context, username string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar_url, bio, last_seen, created_at, updated_at
         FROM users WHERE username=$1`, username)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by login: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, username, email, password_hash, avatar_url, bio, last_seen, created_at, updated_at
         FROM users WHERE id=$1`, id)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by id: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetProfileById(ctx context.Context, profileId int64) (*profile.Profile, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, username, avatar_url, bio, last_seen
         FROM users WHERE id=$1`, profileId)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Username, &p.Avatar, &p.Bio, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by id: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadBio(ctx context.Context, userId int64, bio string) (*profile.Profile, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE users SET bio = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, username, avatar_url, bio, last_seen`,
		userId, bio)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Username, &p.Avatar, &p.Bio, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload bio: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadAvatarUrl(ctx context.Context, userId int64, avatarURL string) (*profile.Profile, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE users SET avatar_url = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, username, avatar_url, bio, last_seen`,
		userId, avatarURL)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Username, &p.Avatar, &p.Bio, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

// Устаревшая часть для чатов
var (
	ErrUserNotFound         = errors.New("User not found")
	ErrEmailAlreadyRegister = errors.New("Email already register")
	ErrLoginAlreadyRegister = errors.New("Login already register")
)

type DepricatedUserRepository struct {
	storage    map[uuid.UUID]*domain.DepricatedUser
	emailIndex map[string]uuid.UUID
	loginIndex map[string]uuid.UUID
	mu         sync.RWMutex
}

func NewDepricatedUserRepository() *DepricatedUserRepository {
	return &DepricatedUserRepository{
		storage:    make(map[uuid.UUID]*domain.DepricatedUser),
		emailIndex: make(map[string]uuid.UUID),
		loginIndex: make(map[string]uuid.UUID),
	}
}

func (repo *DepricatedUserRepository) Create(user *domain.DepricatedUser) error {
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

func (repo *DepricatedUserRepository) GetUserByEmail(email string) (*domain.DepricatedUser, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.emailIndex[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *DepricatedUserRepository) GetUserByLogin(login string) (*domain.DepricatedUser, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	id, ok := repo.loginIndex[login]
	if !ok {
		return nil, ErrUserNotFound
	}
	return repo.storage[id], nil
}

func (repo *DepricatedUserRepository) GetUserByID(user_id uuid.UUID) (*domain.DepricatedUser, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	user, ok := repo.storage[user_id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}
