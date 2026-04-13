package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/sqllog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
)

type UserRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
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
	q := usersql.UploadBirthDate
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, birthDate)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadBirthDate", q, start, err, []any{userId, birthDate})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload bio: %w", err)
	}
	return toDomainProfile(p), nil
}

func NewUserRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*UserRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &UserRepository{db: pool, logger: logger}, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) (_ *domain.User, err error) {
	start := time.Now()

	userModel := toModel(user)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	err = tx.QueryRow(ctx, usersql.ExistsLogin, userModel.Login).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check login: %w", err)
	}
	if exists {
		return nil, domain.ErrLoginAlreadyExists
	}
	err = tx.QueryRow(ctx, usersql.ExistsEmail, userModel.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyExists
	}

	err = tx.QueryRow(ctx, usersql.InsertUser,
		userModel.Login, userModel.FirstName, userModel.LastName, userModel.Email, userModel.PasswordHash,
		userModel.AvatarUrl, userModel.Bio, userModel.BirthDate, userModel.LastSeenAt,
	).Scan(&userModel.Id, &userModel.CreatedAt, &userModel.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "Create", usersql.CreateUserTxDescription, start, err, []any{
		userModel.Login, userModel.FirstName, userModel.LastName, userModel.Email, sqllog.ArgRedacted,
		userModel.AvatarUrl, userModel.Bio, userModel.BirthDate, userModel.LastSeenAt,
	})
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
	q := usersql.GetUserByEmail
	start := time.Now()
	row := r.db.QueryRow(ctx, q, email)

	u := &UserModel{}
	err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByEmail", q, start, err, []any{email})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by email: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	q := usersql.GetUserByLogin
	start := time.Now()
	row := r.db.QueryRow(ctx, q, login)

	u := &UserModel{}
	err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByLogin", q, start, err, []any{login})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by login: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	q := usersql.GetUserByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, id)

	u := &UserModel{}
	err := row.Scan(
		&u.Id, &u.Login, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.AvatarUrl, &u.Bio, &u.BirthDate, &u.LastSeenAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByID", q, start, err, []any{id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get user by id: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetProfileById(ctx context.Context, profileId int64) (*profile.Profile, error) {
	q := usersql.GetProfileByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, profileId)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Email, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetProfileById", q, start, err, []any{profileId})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by id: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadBio(ctx context.Context, userId int64, bio string) (*profile.Profile, error) {
	q := usersql.UploadBio
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, bio)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadBio", q, start, err, []any{userId, bio})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload bio: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadAvatarUrl(ctx context.Context, userId int64, avatarURL string) (*profile.Profile, error) {
	q := usersql.UploadAvatarURL
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, avatarURL)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadAvatarUrl", q, start, err, []any{userId, avatarURL})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadName(ctx context.Context, userId int64, firstName string, lastName *string) (*profile.Profile, error) {
	start := time.Now()
	var row pgx.Row
	var q string

	if lastName == nil {
		q = usersql.UploadNameFirstOnly
		row = r.db.QueryRow(ctx, q, userId, firstName)
	} else {
		q = usersql.UploadNameFull
		row = r.db.QueryRow(ctx, q, userId, firstName, *lastName)
	}

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	if lastName == nil {
		sqllog.LogQuery(ctx, r.log(ctx), "UploadName", q, start, err, []any{userId, firstName})
	} else {
		sqllog.LogQuery(ctx, r.log(ctx), "UploadName", q, start, err, []any{userId, firstName, *lastName})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, profile.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) DeleteUserAvatar(ctx context.Context, userId int64) (*profile.Profile, error) {
	q := usersql.DeleteUserAvatar
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.Login, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteUserAvatar", q, start, err, []any{userId})
	if err != nil {
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

func (r *UserRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
