package chat

import (
	"context"
	"testing"
	"time"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"database/sql"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	chatssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat/sql"
)

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestChatRepository(mock pgxmock.PgxPoolIface) *ChatRepository {
	return &ChatRepository{
		db:     mock,
		logger: zap.NewNop(),
	}
}

func TestChatRepository_GetChatByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		chatID  int64
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
	}{
		{
			name:   "success",
			chatID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				}).AddRow(1, "group", "title", sql.NullString{String: "desc", Valid: true}, 10, sql.NullString{Valid: false}, time.Now(), time.Now())

				m.ExpectQuery(chatssql.GetChatByID).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), chat.Id)
			},
		},
		{
			name:   "not found",
			chatID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetChatByID).
					WithArgs(int64(2)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.ErrorIs(t, err, domain.ErrChatNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)
			chat, err := repo.GetChatByID(ctx, tt.chatID)

			tt.assert(t, chat, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_CreateChat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(1, time.Now(), time.Now())

				m.ExpectQuery(chatssql.InsertChat).
					WithArgs("group", "title", sql.NullString{Valid: false}, int64(10), sql.NullString{Valid: false},).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), chat.Id)
			},
		},
		{
			name: "db_error",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.InsertChat).
					WithArgs("group", "title", sql.NullString{Valid: false}, int64(10), sql.NullString{Valid: false}).
					WillReturnError(errors.New("fail"))
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			input := &domain.Chat{
				Type:    "group",
				Title:   "title",
				OwnerId: 10,
			}

			chat, err := repo.CreateChat(ctx, input)

			tt.assert(t, chat, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_AddMember(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(chatssql.InsertChatMember).
					WithArgs(int64(1), int64(2), "member").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "db_error",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(chatssql.InsertChatMember).
					WithArgs(int64(1), int64(2), "member").
					WillReturnError(errors.New("fail"))
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			err := repo.AddMember(ctx, 1, 2, "member")

			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_DeleteChat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.RepeatableRead,
				})

				m.ExpectExec(chatssql.DeleteMessagesByChatID).
					WithArgs(int64(1)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))

				m.ExpectExec(chatssql.DeleteChatMembersByChatID).
					WithArgs(int64(1)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))

				m.ExpectExec(chatssql.DeleteChatByID).
					WithArgs(int64(1)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))

				m.ExpectCommit()
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error_on_first_query",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectBeginTx(pgx.TxOptions{
					IsoLevel: pgx.RepeatableRead,
				})

				m.ExpectExec(chatssql.DeleteMessagesByChatID).
					WithArgs(int64(1)).
					WillReturnError(errors.New("fail"))

				m.ExpectRollback()
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			err := repo.DeleteChat(ctx, 1)

			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetAllChatsByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		userID  int64
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chats []*domain.Chat, err error)
	}{
		{
			name:   "success_multiple_rows",
			userID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				}).
					AddRow(1, "group", "chat1", sql.NullString{String: "desc", Valid: true}, 10, sql.NullString{Valid: false}, time.Now(), time.Now()).
					AddRow(2, "dialog", "chat2", sql.NullString{String: "desc", Valid: true}, 11, sql.NullString{Valid: false}, time.Now(), time.Now())

				m.ExpectQuery(chatssql.GetAllChatsByUserID).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chats []*domain.Chat, err error) {
				require.NoError(t, err)
				require.Len(t, chats, 2)
			},
		},
		{
			name:   "empty_result",
			userID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				})

				m.ExpectQuery(chatssql.GetAllChatsByUserID).
					WithArgs(int64(2)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chats []*domain.Chat, err error) {
				require.NoError(t, err)
				require.Empty(t, chats)
			},
		},
		{
			name:   "db_error",
			userID: 3,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetAllChatsByUserID).
					WithArgs(int64(3)).
					WillReturnError(errors.New("db fail"))
			},
			assert: func(t *testing.T, chats []*domain.Chat, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			chats, err := repo.GetAllChatsByUserID(ctx, tt.userID)

			tt.assert(t, chats, err)

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetLastMessageOfChat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		chatID  int64
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, msg *domain.Message, err error)
	}{
		{
			name:   "success",
			chatID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "sticker_id",
					"edited", "created_at", "updated_at", "deleted_at",
				}).AddRow(1, 1, 10, "hello", nil, false, time.Now(), time.Now(), nil)

				m.ExpectQuery(chatssql.GetLastMessageOfChat).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, msg *domain.Message, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), msg.Id)
			},
		},
		{
			name:   "no_rows_returns_empty_message",
			chatID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetLastMessageOfChat).
					WithArgs(int64(2)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, msg *domain.Message, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			msg, err := repo.GetLastMessageOfChat(ctx, tt.chatID)

			tt.assert(t, msg, err)

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_DeleteMember(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(chatssql.DeleteChatMember).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "not_found",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(chatssql.DeleteChatMember).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, domain.ErrMemberNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			err := repo.DeleteMember(ctx, 1, 2)

			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}