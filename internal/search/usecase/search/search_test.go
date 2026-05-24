//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=search.go -destination=mock/search_mock.go -package=mock
package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/dto/search"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/search/usecase/search/mock"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestPositiveService_SearchChats(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchChatsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchdto.SearchChatsResponse
		name    string
		args    args
	}{
		{
			name: "Search chats with default limit",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchChats(gomock.Any(), &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup, searchdomain.ChatTypeDialog},
					Limit:    defaultLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchChatsResult{
					Chats: []searchdomain.ChatHit{
						{
							ChatID:             1,
							Title:              "Test Chat",
							Type:               searchdomain.ChatTypeGroup,
							AvatarURL:          strPtr("avatar.jpg"),
							LastMessageAt:      timePtr(now),
							LastMessagePreview: strPtr("Hello!"),
							UnreadCount:        2,
						},
						{
							ChatID:             2,
							Title:              "Another Chat",
							Type:               searchdomain.ChatTypeDialog,
							AvatarURL:          nil,
							LastMessageAt:      nil,
							LastMessagePreview: nil,
							UnreadCount:        0,
						},
					},
					NextBeforeID: 2,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup, searchdomain.ChatTypeDialog},
					Limit:    0,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchChatsResponse{
				Chats: []searchdomain.ChatHit{
					{
						ChatID:             1,
						Title:              "Test Chat",
						Type:               searchdomain.ChatTypeGroup,
						AvatarURL:          strPtr("avatar.jpg"),
						LastMessageAt:      timePtr(now),
						LastMessagePreview: strPtr("Hello!"),
						UnreadCount:        2,
					},
					{
						ChatID:             2,
						Title:              "Another Chat",
						Type:               searchdomain.ChatTypeDialog,
						AvatarURL:          nil,
						LastMessageAt:      nil,
						LastMessagePreview: nil,
						UnreadCount:        0,
					},
				},
				NextBeforeID: 2,
			},
		},
		{
			name: "Search chats with custom limit",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchChats(gomock.Any(), &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup},
					Limit:    10,
					BeforeID: 5,
				}).Return(&searchdomain.SearchChatsResult{
					Chats: []searchdomain.ChatHit{
						{
							ChatID:      3,
							Title:       "Limited Chat",
							Type:        searchdomain.ChatTypeGroup,
							AvatarURL:   nil,
							UnreadCount: 0,
						},
					},
					NextBeforeID: 3,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup},
					Limit:    10,
					BeforeID: 5,
				},
			},
			want: &searchdto.SearchChatsResponse{
				Chats: []searchdomain.ChatHit{
					{
						ChatID:      3,
						Title:       "Limited Chat",
						Type:        searchdomain.ChatTypeGroup,
						AvatarURL:   nil,
						UnreadCount: 0,
					},
				},
				NextBeforeID: 3,
			},
		},
		{
			name: "Search chats with limit exceeding max",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchChats(gomock.Any(), &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "test",
					Kinds:    nil,
					Limit:    maxLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchChatsResult{
					Chats:        []searchdomain.ChatHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "test",
					Limit:    100,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchChatsResponse{
				Chats:        []searchdomain.ChatHit{},
				NextBeforeID: 0,
			},
		},
		{
			name: "Search chats with trimmed query",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchChats(gomock.Any(), &searchdomain.SearchChatsParams{
					UserID:   100,
					Query:    "hello",
					Kinds:    nil,
					Limit:    defaultLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchChatsResult{
					Chats:        []searchdomain.ChatHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "  hello  ",
					Limit:    0,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchChatsResponse{
				Chats:        []searchdomain.ChatHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchChats(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeService_SearchChats(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchChatsRequest
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid user ID (zero)",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID: 0,
					Query:  "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid user ID (negative)",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID: -1,
					Query:  "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Empty query after trim",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID: 100,
					Query:  "   ",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Query too long (exceeds max runes)",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID: 100,
					Query:  string(make([]rune, maxQueryRunes+1)),
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Repository returns error",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchChats(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchChatsRequest{
					UserID: 100,
					Query:  "test",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchChats(tt.args.ctx, tt.args.req)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveService_SearchGlobalChannels(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchGlobalChannelsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchdto.SearchGlobalChannelsResponse
		name    string
		args    args
	}{
		{
			name: "Search global channels successfully",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchGlobalChannels(gomock.Any(), &searchdomain.SearchGlobalChannelsParams{
					UserID:   100,
					Query:    "channel",
					Limit:    defaultLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchGlobalChannelsResult{
					Channels: []searchdomain.GlobalChannelHit{
						{
							ChatID:             1,
							Title:              "Channel 1",
							AvatarURL:          strPtr("channel1.jpg"),
							LastMessageAt:      timePtr(now),
							LastMessagePreview: strPtr("Last message"),
							IsMember:           true,
						},
						{
							ChatID:             2,
							Title:              "Channel 2",
							AvatarURL:          nil,
							LastMessageAt:      nil,
							LastMessagePreview: nil,
							IsMember:           false,
						},
					},
					NextBeforeID: 2,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID:   100,
					Query:    "channel",
					Limit:    0,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchGlobalChannelsResponse{
				Channels: []searchdomain.GlobalChannelHit{
					{
						ChatID:             1,
						Title:              "Channel 1",
						AvatarURL:          strPtr("channel1.jpg"),
						LastMessageAt:      timePtr(now),
						LastMessagePreview: strPtr("Last message"),
						IsMember:           true,
					},
					{
						ChatID:             2,
						Title:              "Channel 2",
						AvatarURL:          nil,
						LastMessageAt:      nil,
						LastMessagePreview: nil,
						IsMember:           false,
					},
				},
				NextBeforeID: 2,
			},
		},
		{
			name: "Search global channels with custom limit and beforeID",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchGlobalChannels(gomock.Any(), &searchdomain.SearchGlobalChannelsParams{
					UserID:   100,
					Query:    "public",
					Limit:    15,
					BeforeID: 10,
				}).Return(&searchdomain.SearchGlobalChannelsResult{
					Channels:     []searchdomain.GlobalChannelHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID:   100,
					Query:    "public",
					Limit:    15,
					BeforeID: 10,
				},
			},
			want: &searchdto.SearchGlobalChannelsResponse{
				Channels:     []searchdomain.GlobalChannelHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchGlobalChannels(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeService_SearchGlobalChannels(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchGlobalChannelsRequest
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid user ID",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID: 0,
					Query:  "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Empty query",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID: 100,
					Query:  "",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Query too long",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID: 100,
					Query:  string(make([]rune, maxQueryRunes+1)),
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchGlobalChannels(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchGlobalChannelsRequest{
					UserID: 100,
					Query:  "test",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchGlobalChannels(tt.args.ctx, tt.args.req)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveService_SearchContacts(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchContactsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchdto.SearchContactsResponse
		name    string
		args    args
	}{
		{
			name: "Search contacts with contacts scope",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchContacts(gomock.Any(), &searchdomain.SearchContactsParams{
					UserID:   100,
					Query:    "john",
					Scope:    searchdomain.ContactScopeContacts,
					Limit:    defaultLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchContactsResult{
					Contacts: []searchdomain.ContactHit{
						{
							UserID:      101,
							DisplayName: "John Doe",
							AvatarURL:   strPtr("john.jpg"),
							Login:       strPtr("john_doe"),
							LastSeenAt:  timePtr(now),
							IsOnline:    true,
						},
						{
							UserID:      102,
							DisplayName: "Johnny",
							AvatarURL:   nil,
							Login:       nil,
							LastSeenAt:  nil,
							IsOnline:    false,
						},
					},
					NextBeforeID: 102,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID:   100,
					Query:    "john",
					Scope:    "contacts",
					Limit:    0,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchContactsResponse{
				Contacts: []searchdomain.ContactHit{
					{
						UserID:      101,
						DisplayName: "John Doe",
						AvatarURL:   strPtr("john.jpg"),
						Login:       strPtr("john_doe"),
						LastSeenAt:  timePtr(now),
						IsOnline:    true,
					},
					{
						UserID:      102,
						DisplayName: "Johnny",
						AvatarURL:   nil,
						Login:       nil,
						LastSeenAt:  nil,
						IsOnline:    false,
					},
				},
				NextBeforeID: 102,
			},
		},
		{
			name: "Search contacts with local scope",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchContacts(gomock.Any(), &searchdomain.SearchContactsParams{
					UserID:   100,
					Query:    "jane",
					Scope:    searchdomain.ContactScopeLocal,
					Limit:    20,
					BeforeID: 0,
				}).Return(&searchdomain.SearchContactsResult{
					Contacts:     []searchdomain.ContactHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID:   100,
					Query:    "jane",
					Scope:    "local",
					Limit:    20,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchContactsResponse{
				Contacts:     []searchdomain.ContactHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchContacts(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeService_SearchContacts(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchContactsRequest
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid user ID",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID: 0,
					Query:  "test",
					Scope:  "contacts",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid scope",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID: 100,
					Query:  "test",
					Scope:  "invalid",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Empty query",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID: 100,
					Query:  "",
					Scope:  "contacts",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchContacts(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchContactsRequest{
					UserID: 100,
					Query:  "test",
					Scope:  "contacts",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchContacts(tt.args.ctx, tt.args.req)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveService_SearchUsers(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchUsersRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchdto.SearchUsersResponse
		name    string
		args    args
	}{
		{
			name: "Search users successfully",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchUsers(gomock.Any(), &searchdomain.SearchUsersParams{
					RequesterID: 100,
					Query:       "alice",
					Limit:       defaultLimit,
					BeforeID:    0,
				}).Return(&searchdomain.SearchUsersResult{
					Users: []searchdomain.ContactHit{
						{
							UserID:      200,
							DisplayName: "Alice Wonder",
							AvatarURL:   strPtr("alice.jpg"),
							Login:       strPtr("alice123"),
							LastSeenAt:  timePtr(now),
							IsOnline:    true,
						},
						{
							UserID:      201,
							DisplayName: "Alice Smith",
							AvatarURL:   nil,
							Login:       strPtr("alice_smith"),
							LastSeenAt:  nil,
							IsOnline:    false,
						},
					},
					NextBeforeID: 201,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 100,
					Query:        "alice",
					Limit:        0,
					BeforeID:     0,
				},
			},
			want: &searchdto.SearchUsersResponse{
				Users: []searchdomain.ContactHit{
					{
						UserID:      200,
						DisplayName: "Alice Wonder",
						AvatarURL:   strPtr("alice.jpg"),
						Login:       strPtr("alice123"),
						LastSeenAt:  timePtr(now),
						IsOnline:    true,
					},
					{
						UserID:      201,
						DisplayName: "Alice Smith",
						AvatarURL:   nil,
						Login:       strPtr("alice_smith"),
						LastSeenAt:  nil,
						IsOnline:    false,
					},
				},
				NextBeforeID: 201,
			},
		},
		{
			name: "Search users with pagination",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchUsers(gomock.Any(), &searchdomain.SearchUsersParams{
					RequesterID: 100,
					Query:       "bob",
					Limit:       25,
					BeforeID:    50,
				}).Return(&searchdomain.SearchUsersResult{
					Users:        []searchdomain.ContactHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 100,
					Query:        "bob",
					Limit:        25,
					BeforeID:     50,
				},
			},
			want: &searchdto.SearchUsersResponse{
				Users:        []searchdomain.ContactHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchUsers(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeService_SearchUsers(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchUsersRequest
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid caller user ID",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 0,
					Query:        "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Empty query",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 100,
					Query:        "",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Query too long",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 100,
					Query:        string(make([]rune, maxQueryRunes+1)),
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchUsers(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchUsersRequest{
					CallerUserID: 100,
					Query:        "test",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchUsers(tt.args.ctx, tt.args.req)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveService_SearchMessagesInChat(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchMessagesInChatRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchdto.SearchMessagesInChatResponse
		name    string
		args    args
	}{
		{
			name: "Search messages in chat successfully",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchMessagesInChat(gomock.Any(), &searchdomain.SearchMessagesInChatParams{
					UserID:   100,
					ChatID:   50,
					Query:    "hello",
					Limit:    defaultLimit,
					BeforeID: 0,
				}).Return(&searchdomain.SearchMessagesInChatResult{
					Messages: []searchdomain.MessageHit{
						{
							MessageID:   1000,
							ChatID:      50,
							SenderID:    101,
							CreatedAt:   now,
							TextPreview: "hello world",
							Highlights: []searchdomain.MessageHighlight{
								{Fragment: "hello"},
							},
						},
						{
							MessageID:   1001,
							ChatID:      50,
							SenderID:    102,
							CreatedAt:   now.Add(time.Hour),
							TextPreview: "say hello",
							Highlights:  []searchdomain.MessageHighlight{},
						},
					},
					NextBeforeID: 1001,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID:   100,
					ChatID:   50,
					Query:    "hello",
					Limit:    0,
					BeforeID: 0,
				},
			},
			want: &searchdto.SearchMessagesInChatResponse{
				Messages: []searchdomain.MessageHit{
					{
						MessageID:   1000,
						ChatID:      50,
						SenderID:    101,
						CreatedAt:   now,
						TextPreview: "hello world",
						Highlights: []searchdomain.MessageHighlight{
							{Fragment: "hello"},
						},
					},
					{
						MessageID:   1001,
						ChatID:      50,
						SenderID:    102,
						CreatedAt:   now.Add(time.Hour),
						TextPreview: "say hello",
						Highlights:  []searchdomain.MessageHighlight{},
					},
				},
				NextBeforeID: 1001,
			},
		},
		{
			name: "Search messages with pagination",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchMessagesInChat(gomock.Any(), &searchdomain.SearchMessagesInChatParams{
					UserID:   100,
					ChatID:   50,
					Query:    "keyword",
					Limit:    30,
					BeforeID: 500,
				}).Return(&searchdomain.SearchMessagesInChatResult{
					Messages:     []searchdomain.MessageHit{},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID:   100,
					ChatID:   50,
					Query:    "keyword",
					Limit:    30,
					BeforeID: 500,
				},
			},
			want: &searchdto.SearchMessagesInChatResponse{
				Messages:     []searchdomain.MessageHit{},
				NextBeforeID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchMessagesInChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeService_SearchMessagesInChat(t *testing.T) {
	type fields struct {
		repo *mock.MockSearchRepository
	}

	type args struct {
		ctx context.Context
		req *searchdto.SearchMessagesInChatRequest
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid user ID",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID: 0,
					ChatID: 50,
					Query:  "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Invalid chat ID",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID: 100,
					ChatID: 0,
					Query:  "test",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Empty query",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID: 100,
					ChatID: 50,
					Query:  "",
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Query too long",
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID: 100,
					ChatID: 50,
					Query:  string(make([]rune, maxQueryRunes+1)),
				},
			},
			wantErr: searchdomain.ErrInvalidInput,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.repo.EXPECT().SearchMessagesInChat(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchdto.SearchMessagesInChatRequest{
					UserID: 100,
					ChatID: 50,
					Query:  "test",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo: mock.NewMockSearchRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &Service{
				repo: f.repo,
			}

			result, err := s.SearchMessagesInChat(tt.args.ctx, tt.args.req)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int32
		want  int32
	}{
		{"Zero limit", 0, defaultLimit},
		{"Negative limit", -5, defaultLimit},
		{"Limit below max", 10, 10},
		{"Limit equal to max", maxLimit, maxLimit},
		{"Limit above max", 100, maxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampLimit(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{"Valid query", "hello world", "hello world", nil},
		{"Query with spaces", "  hello  ", "hello", nil},
		{"Empty query", "", "", searchdomain.ErrInvalidInput},
		{"Only spaces", "   ", "", searchdomain.ErrInvalidInput},
		{"Max length query", string(make([]rune, maxQueryRunes)), string(make([]rune, maxQueryRunes)), nil},
		{"Too long query", string(make([]rune, maxQueryRunes+1)), "", searchdomain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeQuery(tt.input)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				require.Empty(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
