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
