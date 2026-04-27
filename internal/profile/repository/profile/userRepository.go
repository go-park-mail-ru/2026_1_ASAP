package profile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/profile"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/repository/profile/sql"
)

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

type UserRepository struct {
	db     dbPool
	logger *zap.Logger
}

func (r *UserRepository) GetProfileIdByLogin(ctx context.Context, login string) (int64, error) {
	q := usersql.GetProfileIDByLogin
	start := time.Now()
	row := r.db.QueryRow(ctx, q, login)

	var userID int64
	err := row.Scan(&userID)
	sqllog.LogQuery(ctx, r.log(ctx), "GetProfileIdByLogin", q, start, err, []any{login})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, pdomain.ErrNotFound
		}
		return 0, fmt.Errorf("userRepository failed get profile id by login: %w", err)
	}
	return userID, nil
}

func (r *UserRepository) UploadBirthDate(ctx context.Context, userId int64, birthDate *time.Time) (*pdomain.Profile, error) {
	q := usersql.UploadBirthDate
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, birthDate)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadBirthDate", q, start, err, []any{userId, birthDate})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
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

func (r *UserRepository) GetProfileById(ctx context.Context, profileId int64) (*pdomain.Profile, error) {
	q := usersql.GetProfileByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, profileId)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetProfileById", q, start, err, []any{profileId})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed get profile by id: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadBio(ctx context.Context, userId int64, bio string) (*pdomain.Profile, error) {
	q := usersql.UploadBio
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, bio)
	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadBio", q, start, err, []any{userId, bio})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload bio: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadAvatarUrl(ctx context.Context, userId int64, avatarURL string) (*pdomain.Profile, error) {
	q := usersql.UploadAvatarURL
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId, avatarURL)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "UploadAvatarUrl", q, start, err, []any{userId, avatarURL})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) UploadName(ctx context.Context, userId int64, firstName string, lastName *string) (*pdomain.Profile, error) {
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
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	if lastName == nil {
		sqllog.LogQuery(ctx, r.log(ctx), "UploadName", q, start, err, []any{userId, firstName})
	} else {
		sqllog.LogQuery(ctx, r.log(ctx), "UploadName", q, start, err, []any{userId, firstName, *lastName})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
		}
		return nil, fmt.Errorf("userRepository failed upload avatar url: %w", err)
	}
	return toDomainProfile(p), nil
}

func (r *UserRepository) DeleteUserAvatar(ctx context.Context, userId int64) (*pdomain.Profile, error) {
	q := usersql.DeleteUserAvatar
	start := time.Now()
	row := r.db.QueryRow(ctx, q, userId)

	p := &ProfileModel{}
	err := row.Scan(
		&p.UserId, &p.FirstName, &p.LastName, &p.Avatar, &p.Bio, &p.BirthDate, &p.LastSeen,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteUserAvatar", q, start, err, []any{userId})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pdomain.ErrNotFound
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
