package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	profiledomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	userdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"github.com/jackc/pgx/v5"
)

func ptr[T any](v T) *T {
	return &v
}

func newTestUserRepository(mock pgxmock.PgxPoolIface) *UserRepository {
	return &UserRepository{db: mock, logger: zap.NewNop()}
}

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newUserRow(t *testing.T) *pgxmock.Rows {
	t.Helper()
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"id", "login", "first_name", "last_name", "email", "password_hash",
		"avatar_url", "bio", "birth_date", "last_seen", "created_at", "updated_at",
	}).AddRow(
		int64(10), "alice", "Ann", sql.NullString{String: "Smith", Valid: true},
		"alice@example.com", "secret-hash",
		sql.NullString{String: "https://a", Valid: true},
		sql.NullString{String: "hi", Valid: true},
		sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		sql.NullTime{Time: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		created, updated,
	)
}

func newProfileRowNoEmail(t *testing.T) *pgxmock.Rows {
	t.Helper()
	return pgxmock.NewRows([]string{
		"id", "login", "first_name", "last_name", "avatar_url", "bio", "birth_date", "last_seen",
	}).AddRow(
		int64(10), "alice", "Ann", sql.NullString{String: "Smith", Valid: true},
		sql.NullString{String: "https://a", Valid: true},
		sql.NullString{String: "bio", Valid: true},
		sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		sql.NullTime{Time: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	)
}

func newProfileRowWithEmail(t *testing.T) *pgxmock.Rows {
	t.Helper()
	return pgxmock.NewRows([]string{
		"id", "login", "first_name", "last_name", "email", "avatar_url", "bio", "birth_date", "last_seen",
	}).AddRow(
		int64(10), "alice", "Ann", sql.NullString{String: "Smith", Valid: true},
		"alice@example.com",
		sql.NullString{String: "https://a", Valid: true},
		sql.NullString{String: "bio", Valid: true},
		sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		sql.NullTime{Time: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	)
}

func newBaseUserForCreate() *userdomain.User {
	return &userdomain.User{
		Login:        "newuser",
		FirstName:    "N",
		LastName:     ptr("L"),
		Email:        "new@example.com",
		PasswordHash: "hash",
		AvatarUrl:    ptr("https://av"),
		Bio:          ptr("b"),
		BirthDate:    ptr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
		LastSeenAt:   ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
}

func TestUserRepository_GetUserByEmail_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		email      string
		prepare    func(t *testing.T, m pgxmock.PgxPoolIface)
		assertUser func(t *testing.T, u *userdomain.User)
	}{
		{
			name:  "returns_user",
			email: "alice@example.com",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByEmail).WithArgs("alice@example.com").WillReturnRows(newUserRow(t))
			},
			assertUser: func(t *testing.T, u *userdomain.User) {
				t.Helper()
				require.Equal(t, int64(10), u.Id)
				require.Equal(t, "alice", u.Login)
				require.Equal(t, "alice@example.com", u.Email)
				require.Equal(t, "secret-hash", u.PasswordHash)
				require.Equal(t, ptr("Smith"), u.LastName)
				require.Equal(t, ptr("https://a"), u.AvatarUrl)
				require.Equal(t, ptr("hi"), u.Bio)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.GetUserByEmail(ctx, tt.email)
			require.NoError(t, err)
			tt.assertUser(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByEmail_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		email   string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
	}{
		{
			name:  "not_found",
			email: "missing@example.com",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByEmail).WithArgs("missing@example.com").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrNotFound)
			},
		},
		{
			name:  "db_error",
			email: "x@y.z",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByEmail).WithArgs("x@y.z").WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "userRepository failed get user by email")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetUserByEmail(ctx, tt.email)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByLogin_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		login   string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:  "returns_user",
			login: "alice",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("alice").WillReturnRows(newUserRow(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.GetUserByLogin(ctx, tt.login)
			require.NoError(t, err)
			require.Equal(t, "alice", got.Login)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByLogin_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		login   string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
	}{
		{
			name:  "not_found",
			login: "nobody",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("nobody").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrNotFound)
			},
		},
		{
			name:  "db_error",
			login: "alice",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("alice").WillReturnError(errors.New("fail"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "userRepository failed get user by login")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetUserByLogin(ctx, tt.login)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByID_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		id      int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name: "returns_user",
			id:   10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByID).WithArgs(int64(10)).WillReturnRows(newUserRow(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.GetUserByID(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, int64(10), got.Id)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByID_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		id      int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name: "not_found",
			id:   99,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByID).WithArgs(int64(99)).WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetUserByID(ctx, tt.id)
			require.ErrorIs(t, err, userdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileById_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		profileID  int64
		prepare    func(t *testing.T, m pgxmock.PgxPoolIface)
		assertProf func(t *testing.T, p *profiledomain.Profile)
	}{
		{
			name:      "returns_profile",
			profileID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileByID).WithArgs(int64(10)).WillReturnRows(newProfileRowWithEmail(t))
			},
			assertProf: func(t *testing.T, p *profiledomain.Profile) {
				t.Helper()
				require.Equal(t, int64(10), p.UserId)
				require.Equal(t, "alice@example.com", p.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.GetProfileById(ctx, tt.profileID)
			require.NoError(t, err)
			tt.assertProf(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileById_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		profileID int64
		prepare   func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:      "not_found",
			profileID: 1,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileByID).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetProfileById(ctx, tt.profileID)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileIdByLogin_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		login   string
		wantID  int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "returns_id",
			login:  "alice",
			wantID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("alice").WillReturnRows(newUserRow(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			id, err := repo.GetProfileIdByLogin(ctx, tt.login)
			require.NoError(t, err)
			require.Equal(t, tt.wantID, id)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileIdByLogin_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		login   string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:  "not_found",
			login: "ghost",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("ghost").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			id, err := repo.GetProfileIdByLogin(ctx, tt.login)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.Zero(t, id)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBio_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		bio     string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "updates_bio",
			userID: 10,
			bio:    "new bio",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBio).WithArgs(int64(10), "new bio").WillReturnRows(newProfileRowNoEmail(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.UploadBio(ctx, tt.userID, tt.bio)
			require.NoError(t, err)
			require.Equal(t, "bio", *got.Bio)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBio_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		bio     string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "not_found",
			userID: 1,
			bio:    "x",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBio).WithArgs(int64(1), "x").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadBio(ctx, tt.userID, tt.bio)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBirthDate_Positive(t *testing.T) {
	ctx := context.Background()
	bd := time.Date(1991, 2, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		userID  int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "updates_birth_date",
			userID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBirthDate).WithArgs(int64(10), &bd).WillReturnRows(newProfileRowNoEmail(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.UploadBirthDate(ctx, tt.userID, &bd)
			require.NoError(t, err)
			require.Equal(t, int64(10), got.UserId)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBirthDate_Negative(t *testing.T) {
	ctx := context.Background()
	bd := time.Date(1991, 2, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		userID  int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "not_found",
			userID: 7,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBirthDate).WithArgs(int64(7), &bd).WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadBirthDate(ctx, tt.userID, &bd)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadAvatarUrl_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		url     string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "updates_avatar",
			userID: 10,
			url:    "https://new",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadAvatarURL).WithArgs(int64(10), "https://new").WillReturnRows(newProfileRowNoEmail(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.UploadAvatarUrl(ctx, tt.userID, tt.url)
			require.NoError(t, err)
			require.Equal(t, ptr("https://a"), got.Avatar)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadAvatarUrl_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		url     string
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "not_found",
			userID: 2,
			url:    "https://x",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadAvatarURL).WithArgs(int64(2), "https://x").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadAvatarUrl(ctx, tt.userID, tt.url)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadName_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		lastName *string
		wantLast *string
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:     "first_name_only",
			lastName: nil,
			wantLast: ptr("Smith"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFirstOnly).WithArgs(int64(10), "Bob").WillReturnRows(newProfileRowNoEmail(t))
			},
		},
		{
			name:     "first_and_last_name",
			lastName: ptr("Jones"),
			wantLast: ptr("Smith"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFull).WithArgs(int64(10), "Bob", "Jones").WillReturnRows(newProfileRowNoEmail(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.UploadName(ctx, 10, "Bob", tt.lastName)
			require.NoError(t, err)
			require.Equal(t, tt.wantLast, got.LastName)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadName_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		userID   int64
		lastName *string
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:     "first_only_not_found",
			userID:   3,
			lastName: nil,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFirstOnly).WithArgs(int64(3), "Bob").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
		{
			name:     "full_name_not_found",
			userID:   4,
			lastName: ptr("Jones"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFull).WithArgs(int64(4), "Bob", "Jones").WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadName(ctx, tt.userID, "Bob", tt.lastName)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_DeleteUserAvatar_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "clears_avatar",
			userID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.DeleteUserAvatar).WithArgs(int64(10)).WillReturnRows(newProfileRowNoEmail(t))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.DeleteUserAvatar(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, int64(10), got.UserId)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_DeleteUserAvatar_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		userID  int64
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
	}{
		{
			name:   "not_found",
			userID: 5,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.DeleteUserAvatar).WithArgs(int64(5)).WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.DeleteUserAvatar(ctx, tt.userID)
			require.ErrorIs(t, err, profiledomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Create_Positive(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		user    func() *userdomain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User)
		assert  func(t *testing.T, got *userdomain.User)
	}{
		{
			name: "success",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.Serializable,
				})
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "N", sql.NullString{String: "L", Valid: true}, "new@example.com", "hash",
					sql.NullString{String: "https://av", Valid: true},
					sql.NullString{String: "b", Valid: true},
					null.TimePtrToNullTime(u.BirthDate),
					null.TimePtrToNullTime(u.LastSeenAt),
				).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(42), created, updated))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *userdomain.User) {
				t.Helper()
				require.Equal(t, int64(42), got.Id)
				require.Equal(t, "newuser", got.Login)
				require.Equal(t, created, got.CreatedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.user()
			mock := newPGMock(t)
			tt.prepare(t, mock, u)
			repo := newTestUserRepository(mock)

			got, err := repo.Create(ctx, u)
			require.NoError(t, err)
			tt.assert(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Create_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		user    func() *userdomain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User)
		assert  func(t *testing.T, got *userdomain.User, err error)
	}{
		{
			name: "login_already_exists_exists_query",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				_ = u
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.Serializable,
				})
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *userdomain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrLoginAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "email_already_exists_exists_query",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				_ = u
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.Serializable,
				})
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *userdomain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrEmailAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "unique_login_pg_constraint",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.Serializable,
				})
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "N", sql.NullString{String: "L", Valid: true}, "new@example.com", "hash",
					sql.NullString{String: "https://av", Valid: true},
					sql.NullString{String: "b", Valid: true},
					null.TimePtrToNullTime(u.BirthDate),
					null.TimePtrToNullTime(u.LastSeenAt),
				).WillReturnError(&pgconn.PgError{ConstraintName: "users_login_key"})
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *userdomain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrLoginAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "unique_email_pg_constraint",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.Serializable,
				})
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "N", sql.NullString{String: "L", Valid: true}, "new@example.com", "hash",
					sql.NullString{String: "https://av", Valid: true},
					sql.NullString{String: "b", Valid: true},
					null.TimePtrToNullTime(u.BirthDate),
					null.TimePtrToNullTime(u.LastSeenAt),
				).WillReturnError(&pgconn.PgError{ConstraintName: "users_email_key"})
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *userdomain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, userdomain.ErrEmailAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "begin_error",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *userdomain.User) {
				_ = u
				m.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable,}).WillReturnError(errors.New("no tx"))
			},
			assert: func(t *testing.T, got *userdomain.User, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "begin tx")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.user()
			mock := newPGMock(t)
			tt.prepare(t, mock, u)
			repo := newTestUserRepository(mock)

			got, err := repo.Create(ctx, u)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
