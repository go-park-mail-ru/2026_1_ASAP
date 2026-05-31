package profile

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/repository/profile/sql"
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

func newProfileRow(t *testing.T) *pgxmock.Rows {
	t.Helper()
	return pgxmock.NewRows([]string{
		"user_id", "first_name", "last_name", "avatar_url", "bio", "birth_date", "last_seen",
	}).AddRow(
		int64(10),
		"Ann",
		sql.NullString{String: "Smith", Valid: true},
		sql.NullString{String: "https://a", Valid: true},
		sql.NullString{String: "bio", Valid: true},
		sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		sql.NullTime{Time: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	)
}

func TestUserRepository_Create_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare   func(t *testing.T, m pgxmock.PgxPoolIface)
		name      string
		profileID int64
		firstName string
	}{
		{
			name:      "creates_profile",
			profileID: 10,
			firstName: "Ann",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectExec(usersql.CreateProfile).
					WithArgs(int64(10), "Ann").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			err := repo.Create(ctx, tt.profileID, tt.firstName)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Create_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare   func(t *testing.T, m pgxmock.PgxPoolIface)
		assert    func(t *testing.T, err error)
		name      string
		profileID int64
		firstName string
	}{
		{
			name:      "exec_error",
			profileID: 10,
			firstName: "Ann",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectExec(usersql.CreateProfile).
					WithArgs(int64(10), "Ann").
					WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "userRepository failed create profile")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			err := repo.Create(ctx, tt.profileID, tt.firstName)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileIdByLogin_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		login   string
		wantID  int64
	}{
		{
			name:   "returns_id",
			login:  "alice",
			wantID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileIDByLogin).
					WithArgs("alice").
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(10)))
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, id int64, err error)
		name    string
		login   string
	}{
		{
			name:  "not_found",
			login: "ghost",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileIDByLogin).
					WithArgs("ghost").
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, id int64, err error) {
				t.Helper()
				require.Zero(t, id)
				require.ErrorIs(t, err, pdomain.ErrNotFound)
			},
		},
		{
			name:  "db_error",
			login: "alice",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileIDByLogin).
					WithArgs("alice").
					WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, id int64, err error) {
				t.Helper()
				require.Zero(t, id)
				require.Error(t, err)
				require.ErrorContains(t, err, "userRepository failed get profile id by login")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			id, err := repo.GetProfileIdByLogin(ctx, tt.login)
			tt.assert(t, id, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetProfileById_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare    func(t *testing.T, m pgxmock.PgxPoolIface)
		assertProf func(t *testing.T, p *pdomain.Profile)
		name       string
		profileID  int64
	}{
		{
			name:      "returns_profile",
			profileID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileByID).WithArgs(int64(10)).WillReturnRows(newProfileRow(t))
			},
			assertProf: func(t *testing.T, p *pdomain.Profile) {
				t.Helper()
				require.Equal(t, int64(10), p.UserId)
				require.Equal(t, "Ann", p.FirstName)
				require.Equal(t, ptr("Smith"), p.LastName)
				require.Equal(t, ptr("https://a"), p.Avatar)
				require.Equal(t, ptr("bio"), p.Bio)
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
		prepare   func(t *testing.T, m pgxmock.PgxPoolIface)
		assert    func(t *testing.T, err error)
		name      string
		profileID int64
	}{
		{
			name:      "not_found",
			profileID: 1,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetProfileByID).WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, pdomain.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetProfileById(ctx, tt.profileID)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBio_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		bio     string
		userID  int64
	}{
		{
			name:   "updates_bio",
			userID: 10,
			bio:    "new bio",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBio).WithArgs(int64(10), "new bio").WillReturnRows(newProfileRow(t))
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		bio     string
		userID  int64
	}{
		{
			name:   "not_found",
			userID: 1,
			bio:    "x",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBio).WithArgs(int64(1), "x").WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadBio(ctx, tt.userID, tt.bio)
			require.ErrorIs(t, err, pdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadBirthDate_Positive(t *testing.T) {
	ctx := context.Background()
	bd := time.Date(1991, 2, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		userID  int64
	}{
		{
			name:   "updates_birth_date",
			userID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBirthDate).WithArgs(int64(10), &bd).WillReturnRows(newProfileRow(t))
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		userID  int64
	}{
		{
			name:   "not_found",
			userID: 7,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadBirthDate).WithArgs(int64(7), &bd).WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadBirthDate(ctx, tt.userID, &bd)
			require.ErrorIs(t, err, pdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadAvatarUrl_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		url     string
		userID  int64
	}{
		{
			name:   "updates_avatar",
			userID: 10,
			url:    "https://new",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadAvatarURL).WithArgs(int64(10), "https://new").WillReturnRows(newProfileRow(t))
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		url     string
		userID  int64
	}{
		{
			name:   "not_found",
			userID: 2,
			url:    "https://x",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadAvatarURL).WithArgs(int64(2), "https://x").WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadAvatarUrl(ctx, tt.userID, tt.url)
			require.ErrorIs(t, err, pdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UploadName_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		lastName *string
		wantLast *string
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
	}{
		{
			name:     "first_name_only",
			lastName: nil,
			wantLast: ptr("Smith"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFirstOnly).WithArgs(int64(10), "Bob").WillReturnRows(newProfileRow(t))
			},
		},
		{
			name:     "first_and_last_name",
			lastName: ptr("Jones"),
			wantLast: ptr("Smith"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFull).WithArgs(int64(10), "Bob", "Jones").WillReturnRows(newProfileRow(t))
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
		lastName *string
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		name     string
		userID   int64
	}{
		{
			name:     "first_only_not_found",
			userID:   3,
			lastName: nil,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFirstOnly).WithArgs(int64(3), "Bob").WillReturnError(pgx.ErrNoRows)
			},
		},
		{
			name:     "full_name_not_found",
			userID:   4,
			lastName: ptr("Jones"),
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.UploadNameFull).WithArgs(int64(4), "Bob", "Jones").WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.UploadName(ctx, tt.userID, "Bob", tt.lastName)
			require.ErrorIs(t, err, pdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_DeleteUserAvatar_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		userID  int64
	}{
		{
			name:   "clears_avatar",
			userID: 10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.DeleteUserAvatar).WithArgs(int64(10)).WillReturnRows(newProfileRow(t))
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		userID  int64
	}{
		{
			name:   "not_found",
			userID: 5,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.DeleteUserAvatar).WithArgs(int64(5)).WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.DeleteUserAvatar(ctx, tt.userID)
			require.ErrorIs(t, err, pdomain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
