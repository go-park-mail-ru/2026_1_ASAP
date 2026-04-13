package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func (r *UserRepository) GetProfileIdByLogin(ctx context.Context, login string) (int64, error) {
	user, err := r.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return 0, profile.ErrNotFound
		}

		return 0, fmt.Errorf("get id: %w", err)
	}

	return user.Id, nil
}

func (r *UserRepository) UploadBirthDate(ctx context.Context, userId int64, birthDate *time.Time) (*profile.Profile, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE users SET birth_date = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`,
		userId, birthDate)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload bio: %w", err)
	}
	return toDomainProfile(p), nil
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

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`, userModel.Login).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check login: %w", err)
	}
	if exists {
		return nil, domain.ErrLoginAlreadyExists
	}
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, userModel.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyExists
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO users
        (login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        RETURNING id, created_at, updated_at`,
		userModel.Login, userModel.FirstName, userModel.LastName, userModel.Email, userModel.PasswordHash,
		userModel.AvatarUrl, userModel.Bio, userModel.BirthDate, userModel.LastSeenAt,
	).Scan(&userModel.Id, &userModel.CreatedAt, &userModel.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "users_login_key":
				return nil, domain.ErrLoginAlreadyExists
			case "users_email_key":
				return nil, domain.ErrEmailAlreadyExists
			}
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return toDomain(userModel), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
         FROM users WHERE email=$1`, email)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by email: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
         FROM users WHERE login=$1`, login)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
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
		`SELECT id, login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
         FROM users WHERE id=$1`, id)

	u := &UserModel{}
	if err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
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
		`SELECT id, login, first_name, last_name, email, avatar_url, bio, birth_date, last_seen
         FROM users WHERE id=$1`, profileId)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Email, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
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
		 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`,
		userId, bio)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
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
		 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`,
		userId, avatarURL)

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadName(ctx context.Context, userId int64, firstName string, lastName *string) (*profile.Profile, error) {
	var row pgx.Row

	if lastName == nil {
		row = r.db.QueryRow(ctx,
			`UPDATE users SET first_name = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`,
			userId, firstName)
	} else {
		row = r.db.QueryRow(ctx,
			`UPDATE users SET first_name = $2, last_name = $3, updated_at = now()
		 WHERE id = $1
		 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`,
			userId, firstName, *lastName)
	}

	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) DeleteUserAvatar(ctx context.Context, userId int64) (*profile.Profile, error) {
	row := r.db.QueryRow(ctx,
	`UPDATE users
	 SET avatar_url=NULL, updated_at = now()
	 WHERE id=$1
	 RETURNING id, login, first_name, last_name, avatar_url, bio, birth_date, last_seen`, userId)
	
	p := &ProfileModel{}
	if err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) Close() {
	r.db.Close()
}
