// internal/search/repository/postgres/repo_test.go
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/repository/postgres/sql"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func newPGMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock
}

func newTestRepository(mock pgxmock.PgxPoolIface) *Repository {
	return &Repository{db: mock, logger: zap.NewNop()}
}

func TestPositiveRepository_SearchChats(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	type args struct {
		params *searchdomain.SearchChatsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *searchdomain.SearchChatsResult
		name    string
		args    args
	}{
		{
			name: "Search chats successfully with results",
			args: args{
				params: &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup, searchdomain.ChatTypeDialog},
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "avatar_url", "last_message_preview", "last_message_at",
				}).AddRow(
					int64(1), "group", "Test Chat", sql.NullString{String: "avatar.jpg", Valid: true},
					sql.NullString{String: "Hello!", Valid: true}, sql.NullTime{Time: now, Valid: true},
				).AddRow(
					int64(2), "dialog", "Dialog Chat", sql.NullString{Valid: false},
					sql.NullString{Valid: false}, sql.NullTime{Valid: false},
				)
				m.ExpectQuery(searchsql.SearchChats).WithArgs(
					int64(100), "%test%", []string{"group", "dialog"}, int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchChatsResult{
				Chats: []searchdomain.ChatHit{
					{
						ChatID:             1,
						Type:               "group",
						Title:              "Test Chat",
						AvatarURL:          strPtr("avatar.jpg"),
						LastMessagePreview: strPtr("Hello!"),
						LastMessageAt:      timePtr(now),
					},
					{
						ChatID:             2,
						Type:               "dialog",
						Title:              "Dialog Chat",
						AvatarURL:          nil,
						LastMessagePreview: nil,
						LastMessageAt:      nil,
					},
				},
				NextBeforeID: 0,
			},
		},
		{
			name: "Search chats with pagination - returns next_before_id",
			args: args{
				params: &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup},
					Limit:    1,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "avatar_url", "last_message_preview", "last_message_at",
				}).AddRow(
					int64(1), "group", "First Chat", sql.NullString{Valid: false},
					sql.NullString{Valid: false}, sql.NullTime{Valid: false},
				).AddRow(
					int64(2), "group", "Second Chat", sql.NullString{Valid: false},
					sql.NullString{Valid: false}, sql.NullTime{Valid: false},
				)
				m.ExpectQuery(searchsql.SearchChats).WithArgs(
					int64(100), "%test%", []string{"group"}, int64(0), 2,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchChatsResult{
				Chats: []searchdomain.ChatHit{
					{
						ChatID:      1,
						Type:        "group",
						Title:       "First Chat",
						AvatarURL:   nil,
						UnreadCount: 0,
					},
				},
				NextBeforeID: 1,
			},
		},
		{
			name: "Search chats empty result",
			args: args{
				params: &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "nonexistent",
					Kinds:    nil,
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "avatar_url", "last_message_preview", "last_message_at",
				})
				m.ExpectQuery(searchsql.SearchChats).WithArgs(
					int64(100), "%nonexistent%", []string{}, int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchChatsResult{
				Chats:        []searchdomain.ChatHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestRepository(mock)

			got, err := repo.SearchChats(ctx, tt.args.params)
			require.NoError(t, err)
			require.Len(t, got.Chats, len(tt.want.Chats))
			if len(tt.want.Chats) > 0 {
				require.Equal(t, tt.want.Chats[0].ChatID, got.Chats[0].ChatID)
			}
			require.Equal(t, tt.want.NextBeforeID, got.NextBeforeID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeRepository_SearchChats(t *testing.T) {
	ctx := context.Background()

	type args struct {
		params *searchdomain.SearchChatsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Nil params",
			args: args{
				params: nil,
			},
			wantErr: "search chats: params is nil",
		},
		{
			name: "Query error",
			args: args{
				params: &searchdomain.SearchChatsParams{
					UserID: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(searchsql.SearchChats).WithArgs(
					int64(100), "%test%", []string{}, int64(0), 11,
				).WillReturnError(errors.New("db error"))
			},
			wantErr: "search chats query",
		},
		{
			name: "Scan error",
			args: args{
				params: &searchdomain.SearchChatsParams{
					UserID: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "type", "title", "avatar_url", "last_message_preview", "last_message_at",
				}).AddRow(
					"invalid", "group", "Title", sql.NullString{}, sql.NullString{}, sql.NullTime{},
				)
				m.ExpectQuery(searchsql.SearchChats).WithArgs(
					int64(100), "%test%", []string{}, int64(0), 11,
				).WillReturnRows(rows)
			},
			wantErr: "search chats scan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestRepository(mock)

			_, err := repo.SearchChats(ctx, tt.args.params)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveRepository_SearchGlobalChannels(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	type args struct {
		params *searchdomain.SearchGlobalChannelsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *searchdomain.SearchGlobalChannelsResult
		name    string
		args    args
	}{
		{
			name: "Search global channels successfully",
			args: args{
				params: &searchdomain.SearchGlobalChannelsParams{
					UserID:   100,
					Query:    "channel",
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "title", "avatar_url", "last_message_preview", "last_message_at", "is_member",
				}).AddRow(
					int64(1), "Channel 1", sql.NullString{String: "avatar.jpg", Valid: true},
					sql.NullString{String: "Last message", Valid: true}, sql.NullTime{Time: now, Valid: true}, true,
				).AddRow(
					int64(2), "Channel 2", sql.NullString{Valid: false},
					sql.NullString{Valid: false}, sql.NullTime{Valid: false}, false,
				)
				m.ExpectQuery(searchsql.SearchGlobalChannels).WithArgs(
					int64(100), "%channel%", int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchGlobalChannelsResult{
				Channels: []searchdomain.GlobalChannelHit{
					{
						ChatID:             1,
						Title:              "Channel 1",
						AvatarURL:          strPtr("avatar.jpg"),
						LastMessagePreview: strPtr("Last message"),
						LastMessageAt:      timePtr(now),
						IsMember:           true,
					},
					{
						ChatID:             2,
						Title:              "Channel 2",
						AvatarURL:          nil,
						LastMessagePreview: nil,
						LastMessageAt:      nil,
						IsMember:           false,
					},
				},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestRepository(mock)

			got, err := repo.SearchGlobalChannels(ctx, tt.args.params)
			require.NoError(t, err)
			require.Len(t, got.Channels, len(tt.want.Channels))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeRepository_SearchGlobalChannels(t *testing.T) {
	ctx := context.Background()

	type args struct {
		params *searchdomain.SearchGlobalChannelsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Nil params",
			args: args{
				params: nil,
			},
			wantErr: "search global channels: params is nil",
		},
		{
			name: "Query error",
			args: args{
				params: &searchdomain.SearchGlobalChannelsParams{
					UserID: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(searchsql.SearchGlobalChannels).WithArgs(
					int64(100), "%test%", int64(0), 11,
				).WillReturnError(errors.New("db error"))
			},
			wantErr: "search global channels query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestRepository(mock)

			_, err := repo.SearchGlobalChannels(ctx, tt.args.params)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveRepository_SearchContacts(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	type args struct {
		params *searchdomain.SearchContactsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *searchdomain.SearchContactsResult
		name    string
		args    args
	}{
		{
			name: "Search contacts with contacts scope",
			args: args{
				params: &searchdomain.SearchContactsParams{
					UserID:   100,
					Query:    "john",
					Scope:    searchdomain.ContactScopeContacts,
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "first_name", "last_name", "login", "avatar", "last_seen",
				}).AddRow(
					int64(101), "John", sql.NullString{String: "Doe", Valid: true},
					sql.NullString{String: "john_doe", Valid: true}, sql.NullString{String: "avatar.jpg", Valid: true},
					sql.NullTime{Time: now, Valid: true},
				).AddRow(
					int64(102), "Johnny", sql.NullString{Valid: false},
					sql.NullString{Valid: false}, sql.NullString{Valid: false},
					sql.NullTime{Valid: false},
				)
				m.ExpectQuery(searchsql.SearchContacts).WithArgs(
					int64(100), "%john%", int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchContactsResult{
				Contacts: []searchdomain.ContactHit{
					{
						UserID:      101,
						DisplayName: "John Doe",
						Login:       strPtr("john_doe"),
						AvatarURL:   strPtr("avatar.jpg"),
						LastSeenAt:  timePtr(now),
					},
					{
						UserID:      102,
						DisplayName: "Johnny",
						Login:       nil,
						AvatarURL:   nil,
						LastSeenAt:  nil,
					},
				},
				NextBeforeID: 0,
			},
		},
		{
			name: "Search contacts with local scope",
			args: args{
				params: &searchdomain.SearchContactsParams{
					UserID:   100,
					Query:    "jane",
					Scope:    searchdomain.ContactScopeLocal,
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "first_name", "last_name", "login", "avatar", "last_seen",
				}).AddRow(
					int64(201), "Jane", sql.NullString{String: "Smith", Valid: true},
					sql.NullString{String: "jane_smith", Valid: true}, sql.NullString{String: "jane.jpg", Valid: true},
					sql.NullTime{Valid: false},
				)
				m.ExpectQuery(searchsql.SearchUsers).WithArgs(
					int64(100), "%jane%", int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchContactsResult{
				Contacts: []searchdomain.ContactHit{
					{
						UserID:      201,
						DisplayName: "Jane Smith",
						Login:       strPtr("jane_smith"),
						AvatarURL:   strPtr("jane.jpg"),
						LastSeenAt:  nil,
					},
				},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestRepository(mock)

			got, err := repo.SearchContacts(ctx, tt.args.params)
			require.NoError(t, err)
			require.Len(t, got.Contacts, len(tt.want.Contacts))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeRepository_SearchContacts(t *testing.T) {
	ctx := context.Background()

	type args struct {
		params *searchdomain.SearchContactsParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Nil params",
			args: args{
				params: nil,
			},
			wantErr: "search contacts: params is nil",
		},
		{
			name: "Invalid scope",
			args: args{
				params: &searchdomain.SearchContactsParams{
					UserID: 100,
					Query:  "test",
					Scope:  "invalid",
				},
			},
			wantErr: searchdomain.ErrInvalidInput.Error(),
		},
		{
			name: "Query error",
			args: args{
				params: &searchdomain.SearchContactsParams{
					UserID: 100,
					Query:  "test",
					Scope:  searchdomain.ContactScopeContacts,
					Limit:  10,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(searchsql.SearchContacts).WithArgs(
					int64(100), "%test%", int64(0), 11,
				).WillReturnError(errors.New("db error"))
			},
			wantErr: "SearchContacts query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestRepository(mock)

			_, err := repo.SearchContacts(ctx, tt.args.params)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveRepository_SearchUsers(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	type args struct {
		params *searchdomain.SearchUsersParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *searchdomain.SearchUsersResult
		name    string
		args    args
	}{
		{
			name: "Search users successfully",
			args: args{
				params: &searchdomain.SearchUsersParams{
					RequesterID: 100,
					Query:       "alice",
					Limit:       10,
					BeforeID:    0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"user_id", "first_name", "last_name", "login", "avatar", "last_seen",
				}).AddRow(
					int64(200), "Alice", sql.NullString{String: "Wonder", Valid: true},
					sql.NullString{String: "alice_w", Valid: true}, sql.NullString{String: "alice.jpg", Valid: true},
					sql.NullTime{Time: now, Valid: true},
				)
				m.ExpectQuery(searchsql.SearchUsers).WithArgs(
					int64(100), "%alice%", int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchUsersResult{
				Users: []searchdomain.ContactHit{
					{
						UserID:      200,
						DisplayName: "Alice Wonder",
						Login:       strPtr("alice_w"),
						AvatarURL:   strPtr("alice.jpg"),
						LastSeenAt:  timePtr(now),
					},
				},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestRepository(mock)

			got, err := repo.SearchUsers(ctx, tt.args.params)
			require.NoError(t, err)
			require.Len(t, got.Users, len(tt.want.Users))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPositiveRepository_SearchMessagesInChat(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	type args struct {
		params *searchdomain.SearchMessagesInChatParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		want    *searchdomain.SearchMessagesInChatResult
		name    string
		args    args
	}{
		{
			name: "Search messages successfully",
			args: args{
				params: &searchdomain.SearchMessagesInChatParams{
					UserID:   100,
					ChatID:   50,
					Query:    "hello",
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "created_at", "rank",
				}).AddRow(
					int64(1000), int64(50), int64(101), sql.NullString{String: "hello world", Valid: true}, now, 0.5,
				).AddRow(
					int64(1001), int64(50), int64(102), sql.NullString{String: "say hello", Valid: true}, now.Add(time.Hour), 0.3,
				)
				m.ExpectQuery(searchsql.SearchMessagesInChat).WithArgs(
					int64(50), "hello", int64(100), int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchMessagesInChatResult{
				Messages: []searchdomain.MessageHit{
					{
						MessageID:   1000,
						ChatID:      50,
						SenderID:    101,
						TextPreview: "hello world",
						CreatedAt:   now,
					},
					{
						MessageID:   1001,
						ChatID:      50,
						SenderID:    102,
						TextPreview: "say hello",
						CreatedAt:   now.Add(time.Hour),
					},
				},
				NextBeforeID: 0,
			},
		},
		{
			name: "Search messages with content null",
			args: args{
				params: &searchdomain.SearchMessagesInChatParams{
					UserID:   100,
					ChatID:   50,
					Query:    "test",
					Limit:    10,
					BeforeID: 0,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "chat_id", "sender_id", "content", "created_at", "rank",
				}).AddRow(
					int64(1002), int64(50), int64(103), sql.NullString{Valid: false}, now, 0.0,
				)
				m.ExpectQuery(searchsql.SearchMessagesInChat).WithArgs(
					int64(50), "test", int64(100), int64(0), 11,
				).WillReturnRows(rows)
			},
			want: &searchdomain.SearchMessagesInChatResult{
				Messages: []searchdomain.MessageHit{
					{
						MessageID:   1002,
						ChatID:      50,
						SenderID:    103,
						TextPreview: "",
						CreatedAt:   now,
					},
				},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			tt.prepare(t, mock)
			repo := newTestRepository(mock)

			got, err := repo.SearchMessagesInChat(ctx, tt.args.params)
			require.NoError(t, err)
			require.Len(t, got.Messages, len(tt.want.Messages))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNegativeRepository_SearchMessagesInChat(t *testing.T) {
	ctx := context.Background()

	type args struct {
		params *searchdomain.SearchMessagesInChatParams
	}

	tests := []struct {
		prepare func(t *testing.T, m pgxmock.PgxPoolIface)
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Nil params",
			args: args{
				params: nil,
			},
			wantErr: "search messages in chat: params is nil",
		},
		{
			name: "Query error",
			args: args{
				params: &searchdomain.SearchMessagesInChatParams{
					UserID: 100,
					ChatID: 50,
					Query:  "test",
					Limit:  10,
				},
			},
			prepare: func(t *testing.T, m pgxmock.PgxPoolIface) {
				m.ExpectQuery(searchsql.SearchMessagesInChat).WithArgs(
					int64(50), "test", int64(100), int64(0), 11,
				).WillReturnError(errors.New("db error"))
			},
			wantErr: "search messages query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newPGMock(t)
			if tt.prepare != nil {
				tt.prepare(t, mock)
			}
			repo := newTestRepository(mock)

			_, err := repo.SearchMessagesInChat(ctx, tt.args.params)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_Close(t *testing.T) {
	mock := newPGMock(t)
	repo := newTestRepository(mock)
	repo.Close()
}

func TestKindsToStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []searchdomain.ChatType
		want  []string
	}{
		{
			name:  "Empty kinds",
			input: []searchdomain.ChatType{},
			want:  []string{},
		},
		{
			name:  "Single kind",
			input: []searchdomain.ChatType{searchdomain.ChatTypeGroup},
			want:  []string{"group"},
		},
		{
			name:  "Multiple kinds",
			input: []searchdomain.ChatType{searchdomain.ChatTypeGroup, searchdomain.ChatTypeDialog},
			want:  []string{"group", "dialog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kindsToStrings(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
