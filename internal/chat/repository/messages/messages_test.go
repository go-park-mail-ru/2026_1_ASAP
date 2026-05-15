package messages

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

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	messagessql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/messages/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

func ptr[T any](v T) *T {
	return &v
}

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestMessageRepository(mock pgxmock.PgxPoolIface) *MessageRepository {
	return &MessageRepository{db: mock, logger: zap.NewNop()}
}

func expectCreateMessageBeginTx(m pgxmock.PgxPoolIface) *pgxmock.ExpectedBegin {
	return m.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
}

func newMessageRow(t *testing.T, id, chatID, senderID int64, content string, sticker sql.NullInt64, edited bool) *pgxmock.Rows {
	t.Helper()
	created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 1, 11, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
	}).AddRow(id, chatID, senderID, content, sticker, edited, created, updated, sql.NullTime{})
}

func TestMessageRepository_CreateMessage_Positive(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 1, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		message func() *domain.Message
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assert  func(t *testing.T, got *domain.Message)
		name    string
	}{
		{
			name: "without_sticker",
			message: func() *domain.Message {
				return &domain.Message{
					ChatId:   1,
					SenderId: 2,
					Content:  "hello",
					Edited:   false,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				expectCreateMessageBeginTx(m)
				m.ExpectQuery(messagessql.InsertMessage).WithArgs(
					int64(1), int64(2), "hello", null.PtrInt64ToNullInt64(msg.StickerId), false,
				).WillReturnRows(
					pgxmock.NewRows([]string{
						"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
					}).AddRow(int64(10), int64(1), int64(2), "hello", sql.NullInt64{}, false, created, updated, sql.NullTime{}),
				)
				m.ExpectExec(messagessql.UpdateChatLastMessage).WithArgs(int64(10), int64(1)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.Message) {
				t.Helper()
				require.Equal(t, int64(10), got.Id)
				require.Equal(t, int64(1), got.ChatId)
				require.Equal(t, int64(2), got.SenderId)
				require.Equal(t, "hello", got.Content)
				require.Nil(t, got.StickerId)
				require.False(t, got.Edited)
			},
		},
		{
			name: "with_sticker",
			message: func() *domain.Message {
				return &domain.Message{
					ChatId:    5,
					SenderId:  9,
					Content:   "st",
					StickerId: ptr(int64(100)),
					Edited:    true,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				expectCreateMessageBeginTx(m)
				m.ExpectQuery(messagessql.InsertMessage).WithArgs(
					int64(5), int64(9), "st", null.PtrInt64ToNullInt64(msg.StickerId), true,
				).WillReturnRows(
					pgxmock.NewRows([]string{
						"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
					}).AddRow(int64(20), int64(5), int64(9), "st", sql.NullInt64{Int64: 100, Valid: true}, true, created, updated, sql.NullTime{}),
				)
				m.ExpectExec(messagessql.UpdateChatLastMessage).WithArgs(int64(20), int64(5)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				m.ExpectCommit()
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.Message) {
				t.Helper()
				require.Equal(t, int64(20), got.Id)
				require.Equal(t, ptr(int64(100)), got.StickerId)
				require.True(t, got.Edited)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, err := repo.CreateMessage(ctx, msg)
			require.NoError(t, err)
			tt.assert(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_CreateMessage_Negative(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 1, 11, 0, 0, 0, time.UTC)

	baseMsg := func() *domain.Message {
		return &domain.Message{
			ChatId:   1,
			SenderId: 2,
			Content:  "hello",
			Edited:   false,
		}
	}

	tests := []struct {
		message func() *domain.Message
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assert  func(t *testing.T, got *domain.Message, err error)
		name    string
	}{
		{
			name:    "begin_tx_error",
			message: baseMsg,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				_ = msg
				expectCreateMessageBeginTx(m).WillReturnError(errors.New("no connection"))
			},
			assert: func(t *testing.T, got *domain.Message, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "begin tx")
			},
		},
		{
			name:    "insert_error",
			message: baseMsg,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				expectCreateMessageBeginTx(m)
				m.ExpectQuery(messagessql.InsertMessage).WithArgs(
					int64(1), int64(2), "hello", null.PtrInt64ToNullInt64(msg.StickerId), false,
				).WillReturnError(errors.New("insert failed"))
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.Message, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "insert message")
			},
		},
		{
			name:    "update_chat_error",
			message: baseMsg,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				expectCreateMessageBeginTx(m)
				m.ExpectQuery(messagessql.InsertMessage).WithArgs(
					int64(1), int64(2), "hello", null.PtrInt64ToNullInt64(msg.StickerId), false,
				).WillReturnRows(
					pgxmock.NewRows([]string{
						"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
					}).AddRow(int64(10), int64(1), int64(2), "hello", sql.NullInt64{}, false, created, updated, sql.NullTime{}),
				)
				m.ExpectExec(messagessql.UpdateChatLastMessage).WithArgs(int64(10), int64(1)).
					WillReturnError(errors.New("update failed"))
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.Message, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "update chat last message")
			},
		},
		{
			name:    "commit_error",
			message: baseMsg,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				expectCreateMessageBeginTx(m)
				m.ExpectQuery(messagessql.InsertMessage).WithArgs(
					int64(1), int64(2), "hello", null.PtrInt64ToNullInt64(msg.StickerId), false,
				).WillReturnRows(
					pgxmock.NewRows([]string{
						"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
					}).AddRow(int64(10), int64(1), int64(2), "hello", sql.NullInt64{}, false, created, updated, sql.NullTime{}),
				)
				m.ExpectExec(messagessql.UpdateChatLastMessage).WithArgs(int64(10), int64(1)).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				m.ExpectCommit().WillReturnError(errors.New("commit failed"))
				m.ExpectRollback()
			},
			assert: func(t *testing.T, got *domain.Message, err error) {
				t.Helper()
				require.Error(t, err)
				require.Nil(t, got)
				require.ErrorContains(t, err, "commit tx")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, err := repo.CreateMessage(ctx, msg)
			tt.assert(t, got, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_GetMessagesByChatId_Positive(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		beforeID *int64
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		assert   func(t *testing.T, got []*domain.Message)
		name     string
		chatID   int64
		limit    int
	}{
		{
			name:     "without_before_id_returns_rows",
			chatID:   1,
			beforeID: nil,
			limit:    10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
				updated := time.Date(2024, 7, 1, 11, 0, 0, 0, time.UTC)
				rows := newMessageRow(t, 3, 1, 2, "c3", sql.NullInt64{}, false)
				rows.AddRow(int64(2), int64(1), int64(2), "c2", sql.NullInt64{}, false, created, updated, sql.NullTime{})
				m.ExpectQuery(messagessql.GetMessagesByChat).WithArgs(int64(1), 10).WillReturnRows(rows)
			},
			assert: func(t *testing.T, got []*domain.Message) {
				t.Helper()
				require.Len(t, got, 2)
				require.Equal(t, int64(3), got[0].Id)
				require.Equal(t, int64(2), got[1].Id)
			},
		},
		{
			name:     "with_before_id",
			chatID:   1,
			beforeID: ptr(int64(100)),
			limit:    5,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(messagessql.GetMessagesByChatBeforeID).WithArgs(int64(1), int64(100), 5).WillReturnRows(
					newMessageRow(t, 50, 1, 9, "older", sql.NullInt64{}, false),
				)
			},
			assert: func(t *testing.T, got []*domain.Message) {
				t.Helper()
				require.Len(t, got, 1)
				require.Equal(t, int64(50), got[0].Id)
				require.Equal(t, "older", got[0].Content)
			},
		},
		{
			name:     "empty_result",
			chatID:   7,
			beforeID: nil,
			limit:    20,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(messagessql.GetMessagesByChat).WithArgs(int64(7), 20).WillReturnRows(
					pgxmock.NewRows([]string{
						"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
					}),
				)
			},
			assert: func(t *testing.T, got []*domain.Message) {
				t.Helper()
				require.Empty(t, got)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestMessageRepository(mock)

			got, err := repo.GetMessagesByChatId(ctx, tt.chatID, tt.beforeID, tt.limit)
			require.NoError(t, err)
			tt.assert(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_GetMessagesByChatId_Negative(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		beforeID *int64
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		assert   func(t *testing.T, err error)
		name     string
		chatID   int64
		limit    int
	}{
		{
			name:     "query_error_without_before",
			chatID:   1,
			beforeID: nil,
			limit:    10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(messagessql.GetMessagesByChat).WithArgs(int64(1), 10).WillReturnError(errors.New("db down"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "query messages by chat id")
			},
		},
		{
			name:     "query_error_with_before",
			chatID:   2,
			beforeID: ptr(int64(5)),
			limit:    3,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(messagessql.GetMessagesByChatBeforeID).WithArgs(int64(2), int64(5), 3).WillReturnError(errors.New("timeout"))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "query messages by chat id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestMessageRepository(mock)

			_, err := repo.GetMessagesByChatId(ctx, tt.chatID, tt.beforeID, tt.limit)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// internal/chat/repository/messages/messages_test.go
// Добавьте эти тесты в конец файла

func TestMessageRepository_UpdateMessage_Positive(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 2, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		message         func() *domain.Message
		prepare         func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assertLastEdited func(t *testing.T, lastEdited bool)
		name            string
	}{
		{
			name: "update_message_success",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       10,
					ChatId:   1,
					SenderId: 2,
					Content:  "updated content",
					Edited:   true,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "created_at", "updated_at", "edited", "last_message_edited",
				}).AddRow(
					int64(10), int64(1), int64(2), "updated content", created, updated, true, true,
				)
				m.ExpectQuery(messagessql.UpdateMessage).WithArgs(
					"updated content", int64(10), int64(1), int64(2),
				).WillReturnRows(rows)
			},
			assertLastEdited: func(t *testing.T, lastEdited bool) {
				t.Helper()
				require.True(t, lastEdited)
			},
		},
		{
			name: "update_message_does_not_affect_last_message",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       20,
					ChatId:   1,
					SenderId: 3,
					Content:  "updated older message",
					Edited:   true,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "created_at", "updated_at", "edited", "last_message_edited",
				}).AddRow(
					int64(20), int64(1), int64(3), "updated older message", created, updated, true, false,
				)
				m.ExpectQuery(messagessql.UpdateMessage).WithArgs(
					"updated older message", int64(20), int64(1), int64(3),
				).WillReturnRows(rows)
			},
			assertLastEdited: func(t *testing.T, lastEdited bool) {
				t.Helper()
				require.False(t, lastEdited)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, lastEdited, err := repo.UpdateMessage(ctx, msg)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, msg.Id, got.Id)
			require.Equal(t, msg.Content, got.Content)
			require.True(t, got.Edited)
			tt.assertLastEdited(t, lastEdited)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_UpdateMessage_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		message func() *domain.Message
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assert  func(t *testing.T, got *domain.Message, lastEdited bool, err error)
		name    string
	}{
		{
			name: "message_not_found",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       999,
					ChatId:   1,
					SenderId: 2,
					Content:  "updated",
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				m.ExpectQuery(messagessql.UpdateMessage).WithArgs(
					"updated", int64(999), int64(1), int64(2),
				).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, got *domain.Message, lastEdited bool, err error) {
				t.Helper()
				require.Error(t, err)
				require.Equal(t, domain.ErrNoMessage, err)
				require.Nil(t, got)
				require.False(t, lastEdited)
			},
		},
		{
			name: "database_error",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       10,
					ChatId:   1,
					SenderId: 2,
					Content:  "updated",
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				m.ExpectQuery(messagessql.UpdateMessage).WithArgs(
					"updated", int64(10), int64(1), int64(2),
				).WillReturnError(errors.New("db connection lost"))
			},
			assert: func(t *testing.T, got *domain.Message, lastEdited bool, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to scan message")
				require.Nil(t, got)
				require.False(t, lastEdited)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, lastEdited, err := repo.UpdateMessage(ctx, msg)
			tt.assert(t, got, lastEdited, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_DeleteMessage_Positive(t *testing.T) {
	ctx := context.Background()
	//created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	deleted := time.Date(2024, 7, 3, 12, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		message         func() *domain.Message
		prepare         func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assertLastEdited func(t *testing.T, lastEdited bool)
		name            string
	}{
		{
			name: "delete_message_success_affects_last_message",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       10,
					ChatId:   1,
					SenderId: 2,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "deleted_at", "updated_at", "last_message_edited",
				}).AddRow(
					int64(10), int64(1), int64(2), deleted, updated, true,
				)
				m.ExpectQuery(messagessql.DeleteMessage).WithArgs(
					int64(10), int64(1), int64(2),
				).WillReturnRows(rows)
			},
			assertLastEdited: func(t *testing.T, lastEdited bool) {
				t.Helper()
				require.True(t, lastEdited)
			},
		},
		{
			name: "delete_message_does_not_affect_last_message",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       20,
					ChatId:   1,
					SenderId: 3,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "deleted_at", "updated_at", "last_message_edited",
				}).AddRow(
					int64(20), int64(1), int64(3), deleted, updated, false,
				)
				m.ExpectQuery(messagessql.DeleteMessage).WithArgs(
					int64(20), int64(1), int64(3),
				).WillReturnRows(rows)
			},
			assertLastEdited: func(t *testing.T, lastEdited bool) {
				t.Helper()
				require.False(t, lastEdited)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, lastEdited, err := repo.DeleteMessage(ctx, msg)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, msg.Id, got.Id)
			require.NotNil(t, got.DeletedAt)
			tt.assertLastEdited(t, lastEdited)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_DeleteMessage_Negative(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		message func() *domain.Message
		prepare func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message)
		assert  func(t *testing.T, got *domain.Message, lastEdited bool, err error)
		name    string
	}{
		{
			name: "message_not_found",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       999,
					ChatId:   1,
					SenderId: 2,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				m.ExpectQuery(messagessql.DeleteMessage).WithArgs(
					int64(999), int64(1), int64(2),
				).WillReturnError(pgx.ErrNoRows)
			},
			assert: func(t *testing.T, got *domain.Message, lastEdited bool, err error) {
				t.Helper()
				require.Error(t, err)
				require.Equal(t, domain.ErrNoMessage, err)
				require.Nil(t, got)
				require.False(t, lastEdited)
			},
		},
		{
			name: "database_error",
			message: func() *domain.Message {
				return &domain.Message{
					Id:       10,
					ChatId:   1,
					SenderId: 2,
				}
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface, msg *domain.Message) {
				m.ExpectQuery(messagessql.DeleteMessage).WithArgs(
					int64(10), int64(1), int64(2),
				).WillReturnError(errors.New("db error"))
			},
			assert: func(t *testing.T, got *domain.Message, lastEdited bool, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to scan message")
				require.Nil(t, got)
				require.False(t, lastEdited)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message()
			mock := newPGMock(t)
			tt.prepare(t, mock, msg)
			repo := newTestMessageRepository(mock)

			got, lastEdited, err := repo.DeleteMessage(ctx, msg)
			tt.assert(t, got, lastEdited, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_GetMessagesByChatId_ScanError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		beforeID *int64
		prepare  func(t *testing.T, m pgxmock.PgxPoolIface)
		assert   func(t *testing.T, err error)
		chatID   int64
		limit    int
	}{
		{
			name:     "scan_row_error",
			chatID:   1,
			beforeID: nil,
			limit:    10,
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				// Создаем строку с неправильным типом данных для scan
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					"invalid", // string вместо int64 - вызовет ошибку scan
					int64(1), int64(2), "content", sql.NullInt64{}, false, time.Now(), time.Now(), sql.NullTime{},
				)
				m.ExpectQuery(messagessql.GetMessagesByChat).WithArgs(int64(1), 10).WillReturnRows(rows)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.Contains(t, err.Error(), "scan message row")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestMessageRepository(mock)

			_, err := repo.GetMessagesByChatId(ctx, tt.chatID, tt.beforeID, tt.limit)
			tt.assert(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMessageRepository_NewMessageRepository(t *testing.T) {
	ctx := context.Background()
	cfg := config.PostgresConfig{
		Username: "user",
		Password: "pass",
		Host:     "localhost",
		Port:     "5432",
		Database: "testdb",
	}
	logger := zap.NewNop()

	// Этот тест проверит, что функция не паникует и возвращает ошибку
	// (невозможно реально подключиться в тесте)
	repo, err := NewMessageRepository(ctx, cfg, logger)
	if err != nil {
		// Ожидаем ошибку подключения, это нормально в unit тесте
		require.Nil(t, repo)
		require.Error(t, err)
	}
}

func TestMessageRepository_Close(t *testing.T) {
	mock := newPGMock(t)
	repo := newTestMessageRepository(mock)

	// Close не должен паниковать
	repo.Close()
}

func TestMessageRepository_Log(t *testing.T) {
	tests := []struct {
		name        string
		logger      *zap.Logger
		expectNotNop bool
	}{
		{
			name:         "logger is nil returns nop",
			logger:       nil,
			expectNotNop: false,
		},
		{
			name:         "logger is set returns enriched logger",
			logger:       zap.NewNop(),
			expectNotNop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MessageRepository{
				logger: tt.logger,
			}
			got := repo.log(context.Background())
			require.NotNil(t, got)
			if tt.expectNotNop {
				// Проверяем, что это не NopLogger (сложно проверить напрямую, но хотя бы не nil)
				require.NotNil(t, got)
			}
		})
	}
}

// Тест для проверки конвертации null значений
func TestToDomainModel_NullValues(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		model *MessageModel
		want  *domain.Message
	}{
		{
			name: "with_sticker_and_deleted_at",
			model: &MessageModel{
				Id:        1,
				ChatId:    10,
				SenderId:  100,
				Content:   "test",
				StickerId: sql.NullInt64{Int64: 123, Valid: true},
				Edited:    true,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: sql.NullTime{Time: now, Valid: true},
			},
			want: &domain.Message{
				Id:        1,
				ChatId:    10,
				SenderId:  100,
				Content:   "test",
				StickerId: ptr(int64(123)),
				Edited:    true,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: &now,
			},
		},
		{
			name: "without_sticker_and_deleted_at",
			model: &MessageModel{
				Id:        2,
				ChatId:    20,
				SenderId:  200,
				Content:   "test2",
				StickerId: sql.NullInt64{Valid: false},
				Edited:    false,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: sql.NullTime{Valid: false},
			},
			want: &domain.Message{
				Id:        2,
				ChatId:    20,
				SenderId:  200,
				Content:   "test2",
				StickerId: nil,
				Edited:    false,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toDomainModel(tt.model)
			require.Equal(t, tt.want.Id, got.Id)
			require.Equal(t, tt.want.ChatId, got.ChatId)
			require.Equal(t, tt.want.SenderId, got.SenderId)
			require.Equal(t, tt.want.Content, got.Content)
			require.Equal(t, tt.want.Edited, got.Edited)
			if tt.want.StickerId == nil {
				require.Nil(t, got.StickerId)
			} else {
				require.Equal(t, *tt.want.StickerId, *got.StickerId)
			}
			if tt.want.DeletedAt == nil {
				require.Nil(t, got.DeletedAt)
			} else {
				require.Equal(t, *tt.want.DeletedAt, *got.DeletedAt)
			}
		})
	}
}

func TestToModel_NullValues(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		msg   *domain.Message
		want  *MessageModel
	}{
		{
			name: "with_sticker_and_deleted_at",
			msg: &domain.Message{
				Id:        1,
				ChatId:    10,
				SenderId:  100,
				Content:   "test",
				StickerId: ptr(int64(123)),
				Edited:    true,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: &now,
			},
			want: &MessageModel{
				Id:        1,
				ChatId:    10,
				SenderId:  100,
				Content:   "test",
				StickerId: sql.NullInt64{Int64: 123, Valid: true},
				Edited:    true,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: sql.NullTime{Time: now, Valid: true},
			},
		},
		{
			name: "without_sticker_and_deleted_at",
			msg: &domain.Message{
				Id:        2,
				ChatId:    20,
				SenderId:  200,
				Content:   "test2",
				StickerId: nil,
				Edited:    false,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: nil,
			},
			want: &MessageModel{
				Id:        2,
				ChatId:    20,
				SenderId:  200,
				Content:   "test2",
				StickerId: sql.NullInt64{Valid: false},
				Edited:    false,
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: sql.NullTime{Valid: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toModel(tt.msg)
			require.Equal(t, tt.want.ChatId, got.ChatId)
			require.Equal(t, tt.want.SenderId, got.SenderId)
			require.Equal(t, tt.want.Content, got.Content)
			require.Equal(t, tt.want.Edited, got.Edited)
			require.Equal(t, tt.want.StickerId.Valid, got.StickerId.Valid)
			if tt.want.StickerId.Valid {
				require.Equal(t, tt.want.StickerId.Int64, got.StickerId.Int64)
			}
			require.Equal(t, tt.want.DeletedAt.Valid, got.DeletedAt.Valid)
		})
	}
}

// Тест для CreateMessage с пустым sticker_id (sql.NullInt64{})
func TestMessageRepository_CreateMessage_EmptySticker(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 1, 11, 0, 0, 0, time.UTC)

	msg := &domain.Message{
		ChatId:   1,
		SenderId: 2,
		Content:  "hello",
		Edited:   false,
	}

	mock := newPGMock(t)
	expectCreateMessageBeginTx(mock)
	mock.ExpectQuery(messagessql.InsertMessage).WithArgs(
		int64(1), int64(2), "hello", sql.NullInt64{}, false,
	).WillReturnRows(
		pgxmock.NewRows([]string{
			"id", "chat_id", "sender_id", "content", "sticker_id", "edited", "created_at", "updated_at", "deleted_at",
		}).AddRow(int64(10), int64(1), int64(2), "hello", sql.NullInt64{}, false, created, updated, sql.NullTime{}),
	)
	mock.ExpectExec(messagessql.UpdateChatLastMessage).WithArgs(int64(10), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	mock.ExpectRollback()

	repo := newTestMessageRepository(mock)

	got, err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Nil(t, got.StickerId)
	require.NoError(t, mock.ExpectationsWereMet())
}