package contacts

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

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	contactssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/contacts/sql"
)

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestContactsRepository(mock pgxmock.PgxPoolIface) *ContactsRepository {
	return &ContactsRepository{
		db:     mock,
		logger: zap.NewNop(),
	}
}

func TestContactsRepository_GetAllContactsByUserID(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, contacts []*domain.Contact, err error)
		name    string
		userID  int64
	}{
		{
			name:   "success_multiple_contacts",
			userID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "first_name", "last_name", "contact_user_id", "contact_avatar_url", "created_at", "updated_at",
				}).
					AddRow(1, "Alice", sql.NullString{String: "Smith", Valid: true}, 2, sql.NullString{Valid: false}, now, now).
					AddRow(1, "Bob", sql.NullString{String: "Johnson", Valid: true}, 3, sql.NullString{String: "avatar.jpg", Valid: true}, now, now)

				m.ExpectQuery(contactssql.GetAllContactsByUserID).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, contacts []*domain.Contact, err error) {
				require.NoError(t, err)
				require.Len(t, contacts, 2)
				require.Equal(t, int64(1), contacts[0].UserID)
				require.Equal(t, "Alice", contacts[0].FirstName)
				require.Equal(t, "Smith", *contacts[0].LastName)
				require.Equal(t, int64(2), contacts[0].ContactUserID)
			},
		},
		{
			name:   "empty_result",
			userID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "first_name", "last_name", "contact_user_id", "contact_avatar_url", "created_at", "updated_at",
				})

				m.ExpectQuery(contactssql.GetAllContactsByUserID).
					WithArgs(int64(2)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, contacts []*domain.Contact, err error) {
				require.NoError(t, err)
				require.Empty(t, contacts)
			},
		},
		{
			name:   "db_error",
			userID: 3,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(contactssql.GetAllContactsByUserID).
					WithArgs(int64(3)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, contacts []*domain.Contact, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get all contacts")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestContactsRepository(mock)

			contacts, err := repo.GetAllContactsByUserID(ctx, tt.userID)

			tt.assert(t, contacts, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestContactsRepository_CreateContact(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		contact *domain.Contact
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, contact *domain.Contact, err error)
		name    string
	}{
		{
			name: "success_with_last_name",
			contact: &domain.Contact{
				UserID:        1,
				ContactUserID: 2,
				FirstName:     "Alice",
				LastName:      stringPtr("Smith"),
			},
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"user_id", "contact_user_id", "first_name", "last_name", "created_at", "updated_at"}).
					AddRow(1, 2, "Alice", sql.NullString{String: "Smith", Valid: true}, now, now)

				m.ExpectQuery(contactssql.InsertContact).
					WithArgs(int64(1), int64(2), "Alice", sql.NullString{String: "Smith", Valid: true}).
					WillReturnRows(rows)

				avatarRows := pgxmock.NewRows([]string{"avatar_url"}).
					AddRow(sql.NullString{String: "avatar.jpg", Valid: true})

				m.ExpectQuery(contactssql.GetUserAvatarURL).
					WithArgs(int64(2)).
					WillReturnRows(avatarRows)
			},
			assert: func(t *testing.T, contact *domain.Contact, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), contact.UserID)
				require.Equal(t, int64(2), contact.ContactUserID)
				require.Equal(t, "Alice", contact.FirstName)
				require.Equal(t, "Smith", *contact.LastName)
				require.Equal(t, "avatar.jpg", *contact.ContactAvatarUrl)
			},
		},
		{
			name: "success_without_last_name",
			contact: &domain.Contact{
				UserID:        1,
				ContactUserID: 3,
				FirstName:     "Bob",
				LastName:      nil,
			},
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"user_id", "contact_user_id", "first_name", "last_name", "created_at", "updated_at"}).
					AddRow(1, 3, "Bob", sql.NullString{Valid: false}, now, now)

				m.ExpectQuery(contactssql.InsertContact).
					WithArgs(int64(1), int64(3), "Bob", sql.NullString{Valid: false}).
					WillReturnRows(rows)

				avatarRows := pgxmock.NewRows([]string{"avatar_url"}).
					AddRow(sql.NullString{Valid: false})

				m.ExpectQuery(contactssql.GetUserAvatarURL).
					WithArgs(int64(3)).
					WillReturnRows(avatarRows)
			},
			assert: func(t *testing.T, contact *domain.Contact, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), contact.UserID)
				require.Equal(t, int64(3), contact.ContactUserID)
				require.Equal(t, "Bob", contact.FirstName)
				require.Nil(t, contact.LastName)
				require.Nil(t, contact.ContactAvatarUrl)
			},
		},
		{
			name: "success_contact_has_no_avatar",
			contact: &domain.Contact{
				UserID:        1,
				ContactUserID: 4,
				FirstName:     "Charlie",
				LastName:      nil,
			},
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"user_id", "contact_user_id", "first_name", "last_name", "created_at", "updated_at"}).
					AddRow(1, 4, "Charlie", sql.NullString{Valid: false}, now, now)

				m.ExpectQuery(contactssql.InsertContact).
					WithArgs(int64(1), int64(4), "Charlie", sql.NullString{Valid: false}).
					WillReturnRows(rows)

				m.ExpectQuery(contactssql.GetUserAvatarURL).
					WithArgs(int64(4)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, contact *domain.Contact, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), contact.UserID)
				require.Equal(t, int64(4), contact.ContactUserID)
				require.Nil(t, contact.ContactAvatarUrl)
			},
		},
		{
			name: "db_error_on_insert",
			contact: &domain.Contact{
				UserID:        1,
				ContactUserID: 2,
				FirstName:     "Alice",
				LastName:      nil,
			},
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(contactssql.InsertContact).
					WithArgs(int64(1), int64(2), "Alice", sql.NullString{Valid: false}).
					WillReturnError(errors.New("insert error"))
			},
			assert: func(t *testing.T, contact *domain.Contact, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create contact")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestContactsRepository(mock)

			contact, err := repo.CreateContact(ctx, tt.contact)

			tt.assert(t, contact, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestContactsRepository_DeleteContact(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare       func(m pgxmock.PgxPoolIface)
		assert        func(t *testing.T, err error)
		name          string
		userID        int64
		contactUserID int64
	}{
		{
			name:          "success",
			userID:        1,
			contactUserID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(contactssql.DeleteContact).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:          "contact_not_found",
			userID:        1,
			contactUserID: 999,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(contactssql.DeleteContact).
					WithArgs(int64(1), int64(999)).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, domain.ErrContactNotFound)
			},
		},
		{
			name:          "db_error",
			userID:        1,
			contactUserID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(contactssql.DeleteContact).
					WithArgs(int64(1), int64(2)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete contact")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestContactsRepository(mock)

			err := repo.DeleteContact(ctx, tt.userID, tt.contactUserID)

			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestContactsRepository_IsContact(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare       func(m pgxmock.PgxPoolIface)
		assert        func(t *testing.T, exists bool, err error)
		name          string
		userID        int64
		contactUserID int64
	}{
		{
			name:          "contact_exists",
			userID:        1,
			contactUserID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)

				m.ExpectQuery(contactssql.IsContact).
					WithArgs(int64(1), int64(2)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, exists bool, err error) {
				require.NoError(t, err)
				require.True(t, exists)
			},
		},
		{
			name:          "contact_does_not_exist",
			userID:        1,
			contactUserID: 999,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)

				m.ExpectQuery(contactssql.IsContact).
					WithArgs(int64(1), int64(999)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, exists bool, err error) {
				require.NoError(t, err)
				require.False(t, exists)
			},
		},
		{
			name:          "db_error",
			userID:        1,
			contactUserID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(contactssql.IsContact).
					WithArgs(int64(1), int64(2)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, exists bool, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestContactsRepository(mock)

			exists, err := repo.IsContact(ctx, tt.userID, tt.contactUserID)

			tt.assert(t, exists, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
