package user

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	usersql "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/user/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

func expectUserCreateBeginTx(m pgxmock.PgxPoolIface) *pgxmock.ExpectedBegin {
	return m.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable})
}

func newUserRow() *pgxmock.Rows {
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"id", "login", "email", "password_hash", "created_at", "updated_at",
	}).AddRow(
		int64(10), "alice", "alice@example.com", "secret-hash", created, updated,
	)
}

func newBaseUserForCreate() *domain.User {
	return &domain.User{
		Login:        "newuser",
		Email:        "new@example.com",
		PasswordHash: "hash",
	}
}

func TestUserRepository_GetUserByEmail_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare    func(t *testing.T, m pgxmock.PgxPoolIface)
		assertUser func(t *testing.T, u *domain.User)
		name       string
		email      string
	}{
		{
			name:  "returns_user",
			email: "alice@example.com",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByEmail).WithArgs("alice@example.com").WillReturnRows(newUserRow())
			},
			assertUser: func(t *testing.T, u *domain.User) {
				t.Helper()
				require.Equal(t, int64(10), u.ID)
				require.Equal(t, "alice", u.Login)
				require.Equal(t, "alice@example.com", u.Email)
				require.Equal(t, "secret-hash", u.PasswordHash)
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
		email   string
	}{
		{
			name:  "not_found",
			email: "missing@example.com",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByEmail).WithArgs("missing@example.com").WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrNotFound)
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
				require.ErrorContains(t, err, "userRepository failed get profile by email")
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		login   string
	}{
		{
			name:  "returns_user",
			login: "alice",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("alice").WillReturnRows(newUserRow())
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
		login   string
	}{
		{
			name:  "not_found",
			login: "nobody",
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByLogin).WithArgs("nobody").WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrNotFound)
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
				require.ErrorContains(t, err, "userRepository failed get profile by login")
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
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		id      int64
	}{
		{
			name: "returns_user",
			id:   10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByID).WithArgs(int64(10)).WillReturnRows(newUserRow())
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
			require.Equal(t, int64(10), got.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByID_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		id      int64
	}{
		{
			name: "not_found",
			id:   99,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByID).WithArgs(int64(99)).WillReturnError(pgx.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetUserByID(ctx, tt.id)
			require.ErrorIs(t, err, domain.ErrNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByVKID_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		vkID    int64
	}{
		{
			name: "returns_user",
			vkID: 777,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByVKID).WithArgs("777").WillReturnRows(newUserRow())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			got, err := repo.GetUserByVKID(ctx, tt.vkID)
			require.NoError(t, err)
			require.Equal(t, "alice", got.Login)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByVKID_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
		vkID    int64
	}{
		{
			name: "not_found",
			vkID: 1,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByVKID).WithArgs("1").WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrNotFound)
			},
		},
		{
			name: "db_error",
			vkID: 2,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(usersql.GetUserByVKID).WithArgs("2").WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "userRepository failed get profile by vk id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestUserRepository(mock)

			_, err := repo.GetUserByVKID(ctx, tt.vkID)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Create_Positive(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		user    func() *domain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User)
		assert  func(t *testing.T, got *domain.User)
		name    string
	}{
		{
			name: "success",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "new@example.com", "hash",
				).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(42), created, updated))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User) {
				t.Helper()
				require.Equal(t, int64(42), got.ID)
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
		user    func() *domain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User)
		assert  func(t *testing.T, got *domain.User, err error)
		name    string
	}{
		{
			name: "login_already_exists_exists_query",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				_ = u
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrLoginAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "email_already_exists_exists_query",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				_ = u
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "unique_login_pg_constraint",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "new@example.com", "hash",
				).WillReturnError(&pgconn.PgError{ConstraintName: "users_login_key"})
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrLoginAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "unique_email_pg_constraint",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "new@example.com", "hash",
				).WillReturnError(&pgconn.PgError{ConstraintName: "users_email_key"})
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "begin_error",
			user: newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				_ = u
				expectUserCreateBeginTx(m).WillReturnError(errors.New("no tx"))
			},
			assert: func(t *testing.T, got *domain.User, err error) {
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

func TestUserRepository_CreateUserByVKID_Positive(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		user    func() *domain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User)
		assert  func(t *testing.T, got *domain.User)
		name    string
		vkID    int64
	}{
		{
			name: "success",
			vkID:  12345,
			user:  newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "new@example.com", "hash",
				).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(42), created, updated))
				m.ExpectExec(usersql.InsertVKAccount).WithArgs(int64(42), "12345", "new@example.com").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User) {
				t.Helper()
				require.Equal(t, int64(42), got.ID)
				require.Equal(t, "new@example.com", got.Email)
			},
		},
		{
			name: "email exists drops email before linking",
			vkID:  12346,
			user:  newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "", "hash",
				).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(43), created, updated))
				m.ExpectExec(usersql.InsertVKAccount).WithArgs(int64(43), "12346", "").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User) {
				t.Helper()
				require.Equal(t, int64(43), got.ID)
				require.Equal(t, "", got.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.user()
			mock := newPGMock(t)
			tt.prepare(t, mock, u)
			repo := newTestUserRepository(mock)

			got, err := repo.CreateUserByVKID(ctx, tt.vkID, u)
			require.NoError(t, err)
			tt.assert(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_CreateUserByVKID_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		user    func() *domain.User
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User)
		assert  func(t *testing.T, got *domain.User, err error)
		name    string
		vkID    int64
	}{
		{
			name: "login_exists",
			vkID:  1,
			user:  newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				_ = u
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(true),
				)
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.ErrorIs(t, err, domain.ErrLoginAlreadyExists)
				require.Nil(t, got)
			},
		},
		{
			name: "insert_vk_link_error",
			vkID:  2,
			user:  newBaseUserForCreate,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, u *domain.User) {
				created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
				updated := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)
				expectUserCreateBeginTx(m)
				m.ExpectQuery(usersql.ExistsLogin).WithArgs("newuser").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.ExistsEmail).WithArgs("new@example.com").WillReturnRows(
					pgxmock.NewRows([]string{"exists"}).AddRow(false),
				)
				m.ExpectQuery(usersql.InsertUser).WithArgs(
					"newuser", "new@example.com", "hash",
				).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(42), created, updated))
				m.ExpectExec(usersql.InsertVKAccount).WithArgs(int64(42), "2", "new@example.com").
					WillReturnError(errors.New("vk link down"))
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.User, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "failed to link vk account")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.user()
			mock := newPGMock(t)
			tt.prepare(t, mock, u)
			repo := newTestUserRepository(mock)

			got, err := repo.CreateUserByVKID(ctx, tt.vkID, u)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
