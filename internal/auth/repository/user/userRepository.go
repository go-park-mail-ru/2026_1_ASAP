package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/user/sql"
)

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type UserRepository struct {
	db     dbPool
	logger *zap.Logger
}

func (r *UserRepository) CreateUserByVKID(ctx context.Context, vkID int64, newUser *domain.User) (*domain.User, error) {
	start := time.Now()
	userModel := toModel(newUser)
	vkUserID := strconv.FormatInt(vkID, 10)

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
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
		userModel.Email = ""
	}

	err = tx.QueryRow(ctx, usersql.InsertUser,
		userModel.Login, userModel.Email, userModel.PasswordHash,
	).Scan(&userModel.ID, &userModel.CreatedAt, &userModel.UpdatedAt)
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
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	_, err = tx.Exec(ctx, usersql.InsertVKAccount, userModel.ID, vkUserID, userModel.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to link vk account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	sqllog.LogQuery(ctx, r.log(ctx), "CreateUserByVKID", usersql.CreateUserByVKIDTxDescription, start, nil, []any{
		userModel.Login, userModel.Email, vkUserID,
	})

	return toDomain(userModel), nil
}

func (r *UserRepository) GetUserByVKID(ctx context.Context, vkid int64) (*domain.User, error) {
	q := usersql.GetUserByVKID
	start := time.Now()
	vkUserID := strconv.FormatInt(vkid, 10)
	row := r.db.QueryRow(ctx, q, vkUserID)

	u := &UserModel{}
	err := row.Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByVKID", q, start, err, []any{vkUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by vk id: %w", err)
	}
	return toDomain(u), nil
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

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
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
		userModel.Login, userModel.Email, userModel.PasswordHash,
	).Scan(&userModel.ID, &userModel.CreatedAt, &userModel.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "Create", usersql.CreateUserTxDescription, start, err, []any{
		userModel.Login, userModel.Email, sqllog.ArgRedacted,
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
		return nil, fmt.Errorf("failed to create profile: %w", err)
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
	err := row.Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByEmail", q, start, err, []any{email})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by email: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	q := usersql.GetUserByLogin
	start := time.Now()
	row := r.db.QueryRow(ctx, q, login)

	u := &UserModel{}
	err := row.Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByLogin", q, start, err, []any{login})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by login: %w", err)
	}
	return toDomain(u), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	q := usersql.GetUserByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, id)

	u := &UserModel{}
	err := row.Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "GetUserByID", q, start, err, []any{id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by id: %w", err)
	}
	return toDomain(u), nil
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
