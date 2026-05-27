package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	chatssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/chat/sql"
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
		name    string
		chatID  int64
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
		name    string
	}{
		{
			name: "success",
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(1, time.Now(), time.Now())

				m.ExpectQuery(chatssql.InsertChat).
					WithArgs("group", "title", sql.NullString{Valid: false}, int64(10), sql.NullString{Valid: false}).
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chats []*domain.Chat, err error)
		name    string
		userID  int64
	}{
		{
			name:   "success_multiple_rows",
			userID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
					"last_read_message_id", "unread_count",
				}).
					AddRow(1, "group", "chat1", sql.NullString{String: "desc", Valid: true}, 10, sql.NullString{Valid: false}, time.Now(), time.Now(), int64(5), int64(3)).
					AddRow(2, "dialog", "chat2", sql.NullString{String: "desc", Valid: true}, 11, sql.NullString{Valid: false}, time.Now(), time.Now(), int64(0), int64(1))

				m.ExpectQuery(chatssql.GetAllChatsByUserID).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chats []*domain.Chat, err error) {
				require.NoError(t, err)
				require.Len(t, chats, 2)
				require.Equal(t, int64(5), chats[0].LastReadMessageID)
				require.Equal(t, int64(3), chats[0].UnreadCount)
				require.Equal(t, int64(1), chats[1].UnreadCount)
			},
		},
		{
			name:   "empty_result",
			userID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
					"last_read_message_id", "unread_count",
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

func TestChatRepository_GetChatMemberUnread(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, lastRead, unread int64, err error)
		name    string
		chatID  int64
		userID  int64
	}{
		{
			name:   "success",
			chatID: 1,
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetChatMemberUnread).
					WithArgs(int64(1), int64(100)).
					WillReturnRows(pgxmock.NewRows([]string{"last_read_message_id", "unread_count"}).
						AddRow(int64(10), int64(2)))
			},
			assert: func(t *testing.T, lastRead, unread int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(10), lastRead)
				require.Equal(t, int64(2), unread)
			},
		},
		{
			name:   "not_member",
			chatID: 2,
			userID: 200,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetChatMemberUnread).
					WithArgs(int64(2), int64(200)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, lastRead, unread int64, err error) {
				require.ErrorIs(t, err, domain.ErrNotMember)
				require.Zero(t, lastRead)
				require.Zero(t, unread)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)
			lastRead, unread, err := repo.GetChatMemberUnread(ctx, tt.chatID, tt.userID)
			tt.assert(t, lastRead, unread, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetLastMessageOfChat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, msg *domain.Message, err error)
		name    string
		chatID  int64
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
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, err error)
		name    string
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

func TestChatRepository_GetLastMessagesOfChats(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, msgs []*domain.Message, err error)
		name    string
		userID  int64
	}{
		{
			name:   "success with multiple messages",
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
				}).
					AddRow(1, 10, 100, "Hello", nil, false, now, now, nil).
					AddRow(2, 20, 101, "Hi", nil, false, now, now, nil)

				m.ExpectQuery(chatssql.GetLastMessagesOfChats).
					WithArgs(int64(100)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, msgs []*domain.Message, err error) {
				require.NoError(t, err)
				require.Len(t, msgs, 2)
				require.Equal(t, int64(1), msgs[0].Id)
				require.Equal(t, int64(2), msgs[1].Id)
			},
		},
		{
			name:   "empty result",
			userID: 200,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
				})

				m.ExpectQuery(chatssql.GetLastMessagesOfChats).
					WithArgs(int64(200)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, msgs []*domain.Message, err error) {
				require.NoError(t, err)
				require.Empty(t, msgs)
			},
		},
		{
			name:   "db error",
			userID: 300,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetLastMessagesOfChats).
					WithArgs(int64(300)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, msgs []*domain.Message, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get last messages")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			msgs, err := repo.GetLastMessagesOfChats(ctx, tt.userID)

			tt.assert(t, msgs, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetChatMembers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, members []int64, err error)
		name    string
		chatID  int64
	}{
		{
			name:   "success with multiple members",
			chatID: 1,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"user_id"}).
					AddRow(100).
					AddRow(101).
					AddRow(102)

				m.ExpectQuery(chatssql.GetChatMembers).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, members []int64, err error) {
				require.NoError(t, err)
				require.Len(t, members, 3)
				require.Contains(t, members, int64(100))
				require.Contains(t, members, int64(101))
				require.Contains(t, members, int64(102))
			},
		},
		{
			name:   "empty result",
			chatID: 2,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"user_id"})

				m.ExpectQuery(chatssql.GetChatMembers).
					WithArgs(int64(2)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, members []int64, err error) {
				require.NoError(t, err)
				require.Empty(t, members)
			},
		},
		{
			name:   "db error",
			chatID: 3,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetChatMembers).
					WithArgs(int64(3)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, members []int64, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get chat members")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			members, err := repo.GetChatMembers(ctx, tt.chatID)

			tt.assert(t, members, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_IsMember(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, isMember bool, err error)
		name    string
		chatID  int64
		userID  int64
	}{
		{
			name:   "user is member",
			chatID: 1,
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)

				m.ExpectQuery(chatssql.IsMember).
					WithArgs(int64(1), int64(100)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, isMember bool, err error) {
				require.NoError(t, err)
				require.True(t, isMember)
			},
		},
		{
			name:   "user is not member",
			chatID: 1,
			userID: 200,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)

				m.ExpectQuery(chatssql.IsMember).
					WithArgs(int64(1), int64(200)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, isMember bool, err error) {
				require.NoError(t, err)
				require.False(t, isMember)
			},
		},
		{
			name:   "db error",
			chatID: 1,
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.IsMember).
					WithArgs(int64(1), int64(100)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, isMember bool, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to check if member exists")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			isMember, err := repo.IsMember(ctx, tt.chatID, tt.userID)

			tt.assert(t, isMember, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetDialogBetweenUsers(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
		name    string
		user1ID int64
		user2ID int64
	}{
		{
			name:    "dialog exists",
			user1ID: 100,
			user2ID: 101,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				}).AddRow(1, "dialog", "", sql.NullString{Valid: false}, 100, sql.NullString{Valid: false}, now, now)

				m.ExpectQuery(chatssql.GetDialogBetweenUsers).
					WithArgs(int64(100), int64(101)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.NoError(t, err)
				require.NotNil(t, chat)
				require.Equal(t, int64(1), chat.Id)
				require.Equal(t, domain.ChatTypeDialog, chat.Type)
			},
		},
		{
			name:    "dialog does not exist - returns nil chat",
			user1ID: 100,
			user2ID: 200,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetDialogBetweenUsers).
					WithArgs(int64(100), int64(200)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {

				require.NoError(t, err)
				require.Nil(t, chat)
			},
		},
		{
			name:    "db error",
			user1ID: 100,
			user2ID: 101,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetDialogBetweenUsers).
					WithArgs(int64(100), int64(101)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get dialog between users")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			chat, err := repo.GetDialogBetweenUsers(ctx, tt.user1ID, tt.user2ID)

			tt.assert(t, chat, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_GetMemberRole(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, role string, err error)
		name    string
		chatID  int64
		userID  int64
	}{
		{
			name:   "get owner role",
			chatID: 1,
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"role"}).AddRow("owner")

				m.ExpectQuery(chatssql.GetMemberRole).
					WithArgs(int64(1), int64(100)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, role string, err error) {
				require.NoError(t, err)
				require.Equal(t, "owner", role)
			},
		},
		{
			name:   "get admin role",
			chatID: 1,
			userID: 101,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"role"}).AddRow("admin")

				m.ExpectQuery(chatssql.GetMemberRole).
					WithArgs(int64(1), int64(101)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, role string, err error) {
				require.NoError(t, err)
				require.Equal(t, "admin", role)
			},
		},
		{
			name:   "get member role",
			chatID: 1,
			userID: 102,
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"role"}).AddRow("member")

				m.ExpectQuery(chatssql.GetMemberRole).
					WithArgs(int64(1), int64(102)).
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, role string, err error) {
				require.NoError(t, err)
				require.Equal(t, "member", role)
			},
		},
		{
			name:   "user not member - returns empty string",
			chatID: 1,
			userID: 999,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetMemberRole).
					WithArgs(int64(1), int64(999)).
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, role string, err error) {
				require.NoError(t, err)
				require.Empty(t, role)
			},
		},
		{
			name:   "db error",
			chatID: 1,
			userID: 100,
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.GetMemberRole).
					WithArgs(int64(1), int64(100)).
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, role string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get user role")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			role, err := repo.GetMemberRole(ctx, tt.userID, tt.chatID)

			tt.assert(t, role, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_UploadAvatarUrl(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		prepare   func(m pgxmock.PgxPoolIface)
		assert    func(t *testing.T, chat *domain.Chat, err error)
		name      string
		avatarURL string
		chatID    int64
	}{
		{
			name:      "success upload avatar url",
			chatID:    1,
			avatarURL: "https://cdn.example.com/avatar.jpg",
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				}).AddRow(1, "group", "Chat", sql.NullString{Valid: false}, 100, sql.NullString{String: "https://cdn.example.com/avatar.jpg", Valid: true}, now, now)

				m.ExpectQuery(chatssql.UpdateChatAvatarURL).
					WithArgs(int64(1), "https://cdn.example.com/avatar.jpg").
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), chat.Id)
				require.Equal(t, "https://cdn.example.com/avatar.jpg", *chat.AvatarUrl)
			},
		},
		{
			name:      "chat not found",
			chatID:    999,
			avatarURL: "https://cdn.example.com/avatar.jpg",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.UpdateChatAvatarURL).
					WithArgs(int64(999), "https://cdn.example.com/avatar.jpg").
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.ErrorIs(t, err, domain.ErrChatNotFound)
			},
		},
		{
			name:      "db error",
			chatID:    1,
			avatarURL: "https://cdn.example.com/avatar.jpg",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.UpdateChatAvatarURL).
					WithArgs(int64(1), "https://cdn.example.com/avatar.jpg").
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "chatRepo failed upload avatar url")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			chat, err := repo.UploadAvatarUrl(ctx, tt.chatID, tt.avatarURL)

			tt.assert(t, chat, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatRepository_UpdateTitle(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		prepare func(m pgxmock.PgxPoolIface)
		assert  func(t *testing.T, chat *domain.Chat, err error)
		name    string
		title   string
		chatID  int64
	}{
		{
			name:   "success update title",
			chatID: 1,
			title:  "New Title",
			prepare: func(m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "description", "owner_id", "avatar_url", "created_at", "updated_at",
				}).AddRow(1, "group", "New Title", sql.NullString{Valid: false}, 100, sql.NullString{Valid: false}, now, now)

				m.ExpectQuery(chatssql.UpdateChatTitle).
					WithArgs(int64(1), "New Title").
					WillReturnRows(rows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(1), chat.Id)
				require.Equal(t, "New Title", chat.Title)
			},
		},
		{
			name:   "chat not found",
			chatID: 999,
			title:  "New Title",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.UpdateChatTitle).
					WithArgs(int64(999), "New Title").
					WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.ErrorIs(t, err, domain.ErrChatNotFound)
			},
		},
		{
			name:   "db error",
			chatID: 1,
			title:  "New Title",
			prepare: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(chatssql.UpdateChatTitle).
					WithArgs(int64(1), "New Title").
					WillReturnError(errors.New("database error"))
			},
			assert: func(t *testing.T, chat *domain.Chat, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "chatRepo failed update title")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(mock)

			repo := newTestChatRepository(mock)

			chat, err := repo.UpdateTitle(ctx, tt.chatID, tt.title)

			tt.assert(t, chat, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
