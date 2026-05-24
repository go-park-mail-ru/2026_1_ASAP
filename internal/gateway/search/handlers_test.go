//go:generate mockgen -destination=mock/search_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1 SearchClient
package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/search/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveGatewaySearchHandler_SearchMessages(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID   int64
		chatID   string
		query    string
		limit    string
		beforeID string
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful search messages",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchMessagesInChat(gomock.Any(), &searchv1.SearchMessagesInChatRequest{
					UserId:   100,
					ChatId:   50,
					Query:    "hello",
					Limit:    20,
					BeforeId: 0,
				}).Return(&searchv1.SearchMessagesInChatResponse{
					Messages: []*searchv1.SearchMessageItem{
						{
							MessageId:   1000,
							ChatId:      50,
							SenderId:    101,
							TextPreview: "hello world",
							CreatedAt:   timestamppb.Now(),
							Highlights: &searchv1.SearchMessageHighlight{
								Fragment: "hello",
							},
						},
					},
					NextBeforeId: 1000,
				}, nil)
			},
			args: args{
				userID:   100,
				chatID:   "50",
				query:    "hello",
				limit:    "20",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/messages", handler.SearchMessages)

			req := httptest.NewRequest(http.MethodGet, "/search/messages?chat_id="+tt.args.chatID+"&q="+tt.args.query+"&limit="+tt.args.limit+"&before_id="+tt.args.beforeID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewaySearchHandler_SearchMessages(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID   interface{}
		chatID   string
		query    string
		limit    string
		beforeID string
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Missing user_id",
			args: args{
				userID: nil,
				chatID: "50",
				query:  "test",
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid chat_id",
			args: args{
				userID: int64(100),
				chatID: "invalid",
				query:  "test",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Empty chat_id",
			args: args{
				userID: int64(100),
				chatID: "0",
				query:  "test",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid input error",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchMessagesInChat(gomock.Any(), &searchv1.SearchMessagesInChatRequest{
					UserId:   100,
					ChatId:   50,
					Query:    "test",
					Limit:    0,
					BeforeId: 0,
				}).Return(nil, grpcerr.New(codes.InvalidArgument, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INVALID_INPUT), "invalid input"))
			},
			args: args{
				userID: int64(100),
				chatID: "50",
				query:  "test",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Not found error",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchMessagesInChat(gomock.Any(), &searchv1.SearchMessagesInChatRequest{
					UserId:   100,
					ChatId:   50,
					Query:    "test",
					Limit:    0,
					BeforeId: 0,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(searchv1.SearchErrorCode_SEARCH_ERROR_NOT_FOUND), "not found"))
			},
			args: args{
				userID: int64(100),
				chatID: "50",
				query:  "test",
			},
			want: http.StatusNotFound,
		},
		{
			name: "Permission denied error",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchMessagesInChat(gomock.Any(), &searchv1.SearchMessagesInChatRequest{
					UserId:   100,
					ChatId:   50,
					Query:    "test",
					Limit:    0,
					BeforeId: 0,
				}).Return(nil, grpcerr.New(codes.PermissionDenied, int32(searchv1.SearchErrorCode_SEARCH_ERROR_PERMISSION_DENIED), "permission denied"))
			},
			args: args{
				userID: int64(100),
				chatID: "50",
				query:  "test",
			},
			want: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/messages", handler.SearchMessages)

			url := "/search/messages?chat_id=" + tt.args.chatID + "&q=" + tt.args.query
			if tt.args.limit != "" {
				url += "&limit=" + tt.args.limit
			}
			if tt.args.beforeID != "" {
				url += "&before_id=" + tt.args.beforeID
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			if tt.args.userID != nil {
				if uid, ok := tt.args.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				}
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewaySearchHandler_SearchUsers(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID   int64
		query    string
		scope    string
		limit    string
		beforeID string
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful search users with contacts scope",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchContacts(gomock.Any(), &searchv1.SearchContactsRequest{
					UserId:   100,
					Query:    "john",
					Scope:    searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS,
					Limit:    20,
					BeforeId: 0,
				}).Return(&searchv1.SearchContactsResponse{
					Contacts: []*searchv1.SearchContactItem{
						{
							UserId:      101,
							DisplayName: "John Doe",
							Login:       strPtr("john_doe"),
							AvatarUrl:   strPtr("avatar.jpg"),
							IsOnline:    true,
							LastSeenAt:  timestamppb.Now(),
						},
					},
					NextBeforeId: 101,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "john",
				scope:    "contacts",
				limit:    "20",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
		{
			name: "Successful search users with local scope",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchContacts(gomock.Any(), &searchv1.SearchContactsRequest{
					UserId:   100,
					Query:    "jane",
					Scope:    searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL,
					Limit:    10,
					BeforeId: 0,
				}).Return(&searchv1.SearchContactsResponse{
					Contacts:     []*searchv1.SearchContactItem{},
					NextBeforeId: 0,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "jane",
				scope:    "",
				limit:    "10",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/users", handler.SearchUsers)

			url := "/search/users?q=" + tt.args.query
			if tt.args.scope != "" {
				url += "&scope=" + tt.args.scope
			}
			if tt.args.limit != "" {
				url += "&limit=" + tt.args.limit
			}
			if tt.args.beforeID != "" {
				url += "&before_id=" + tt.args.beforeID
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewaySearchHandler_SearchUsers(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID interface{}
		query  string
		scope  string
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Missing user_id",
			args: args{
				userID: nil,
				query:  "test",
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid input error",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchContacts(gomock.Any(), &searchv1.SearchContactsRequest{
					UserId:   100,
					Query:    "test",
					Scope:    searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL,
					Limit:    0,
					BeforeId: 0,
				}).Return(nil, grpcerr.New(codes.InvalidArgument, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INVALID_INPUT), "invalid input"))
			},
			args: args{
				userID: int64(100),
				query:  "test",
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/users", handler.SearchUsers)

			url := "/search/users?q=" + tt.args.query
			if tt.args.scope != "" {
				url += "&scope=" + tt.args.scope
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			if tt.args.userID != nil {
				if uid, ok := tt.args.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				}
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewaySearchHandler_SearchChats(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID   int64
		query    string
		chatType string
		limit    string
		beforeID string
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful search chats with group type",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchChats(gomock.Any(), &searchv1.SearchChatsRequest{
					UserId:   100,
					Query:    "team",
					Kinds:    []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
					Limit:    20,
					BeforeId: 0,
				}).Return(&searchv1.SearchChatsResponse{
					Chats: []*searchv1.SearchChatItem{
						{
							ChatId:             1,
							Type:               searchv1.ChatType_CHAT_TYPE_GROUP,
							Title:              "Team Chat",
							AvatarUrl:          strPtr("avatar.jpg"),
							LastMessagePreview: strPtr("Hello!"),
							LastMessageAt:      timestamppb.Now(),
							UnreadCount:        3,
						},
					},
					NextBeforeId: 1,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "team",
				chatType: "group",
				limit:    "20",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
		{
			name: "Successful search chats with dialog type",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchChats(gomock.Any(), &searchv1.SearchChatsRequest{
					UserId:   100,
					Query:    "hello",
					Kinds:    []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
					Limit:    10,
					BeforeId: 0,
				}).Return(&searchv1.SearchChatsResponse{
					Chats:        []*searchv1.SearchChatItem{},
					NextBeforeId: 0,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "hello",
				chatType: "dialog",
				limit:    "10",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
		{
			name: "Successful search chats with channel type (uses SearchGlobalChannels)",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchGlobalChannels(gomock.Any(), &searchv1.SearchGlobalChannelsRequest{
					UserId:   100,
					Query:    "public",
					Limit:    15,
					BeforeId: 0,
				}).Return(&searchv1.SearchGlobalChannelsResponse{
					Channels: []*searchv1.SearchGlobalChannelItem{
						{
							ChatId:             10,
							Title:              "Public Channel",
							AvatarUrl:          strPtr("channel.jpg"),
							LastMessagePreview: strPtr("Welcome!"),
							LastMessageAt:      timestamppb.Now(),
							IsMember:           false,
						},
					},
					NextBeforeId: 10,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "public",
				chatType: "channel",
				limit:    "15",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
		{
			name: "Successful search chats with multiple types",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchChats(gomock.Any(), &searchv1.SearchChatsRequest{
					UserId:   100,
					Query:    "test",
					Kinds:    []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP, searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
					Limit:    20,
					BeforeId: 0,
				}).Return(&searchv1.SearchChatsResponse{
					Chats:        []*searchv1.SearchChatItem{},
					NextBeforeId: 0,
				}, nil)
			},
			args: args{
				userID:   100,
				query:    "test",
				chatType: "group,dialog",
				limit:    "20",
				beforeID: "0",
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/chats", handler.SearchChats)

			url := "/search/chats?q=" + tt.args.query + "&type=" + tt.args.chatType
			if tt.args.limit != "" {
				url += "&limit=" + tt.args.limit
			}
			if tt.args.beforeID != "" {
				url += "&before_id=" + tt.args.beforeID
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewaySearchHandler_SearchChats(t *testing.T) {
	type fields struct {
		searchClient *mock.MockSearchClient
	}

	type args struct {
		userID   interface{}
		query    string
		chatType string
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Missing user_id",
			args: args{
				userID: nil,
				query:  "test",
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid chat type",
			args: args{
				userID:   int64(100),
				query:    "test",
				chatType: "invalid",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid input error",
			prepare: func(f *fields) {
				f.searchClient.EXPECT().SearchChats(gomock.Any(), &searchv1.SearchChatsRequest{
					UserId:   100,
					Query:    "test",
					Kinds:    nil,
					Limit:    0,
					BeforeId: 0,
				}).Return(nil, grpcerr.New(codes.InvalidArgument, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INVALID_INPUT), "invalid input"))
			},
			args: args{
				userID: int64(100),
				query:  "test",
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchClient: mock.NewMockSearchClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewaySearchHandler{
				Search: f.searchClient,
			}

			r := chi.NewRouter()
			r.Get("/search/chats", handler.SearchChats)

			url := "/search/chats?q=" + tt.args.query
			if tt.args.chatType != "" {
				url += "&type=" + tt.args.chatType
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			if tt.args.userID != nil {
				if uid, ok := tt.args.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				}
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestParseChatKinds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []searchv1.SearchChatKind
		wantErr bool
	}{
		{
			name:  "Empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "Single dialog",
			input: "dialog",
			want:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
		},
		{
			name:  "Single group",
			input: "group",
			want:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
		},
		{
			name:  "Single channel",
			input: "channel",
			want:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL},
		},
		{
			name:  "Multiple types",
			input: "group,dialog",
			want:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP, searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
		},
		{
			name:  "With spaces",
			input: " group , dialog ",
			want:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP, searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
		},
		{
			name:    "Invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChatKinds(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tt.want, got)
			}
		})
	}
}

func TestMapSearchScope(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  searchv1.SearchContactScope
	}{
		{
			name:  "Contacts scope",
			input: "contacts",
			want:  searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS,
		},
		{
			name:  "Empty scope defaults to local",
			input: "",
			want:  searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL,
		},
		{
			name:  "Invalid scope defaults to local",
			input: "invalid",
			want:  searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSearchScope(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMapSearchChatType(t *testing.T) {
	tests := []struct {
		name  string
		input searchv1.ChatType
		want  string
	}{
		{
			name:  "Group type",
			input: searchv1.ChatType_CHAT_TYPE_GROUP,
			want:  "group",
		},
		{
			name:  "Channel type",
			input: searchv1.ChatType_CHAT_TYPE_CHANNEL,
			want:  "channel",
		},
		{
			name:  "Dialog type",
			input: searchv1.ChatType_CHAT_TYPE_DIALOG,
			want:  "dialog",
		},
		{
			name:  "Unspecified defaults to dialog",
			input: searchv1.ChatType_CHAT_TYPE_UNSPECIFIED,
			want:  "dialog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSearchChatType(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsOnlyChannelKindFilter(t *testing.T) {
	tests := []struct {
		name  string
		input []searchv1.SearchChatKind
		want  bool
	}{
		{
			name:  "Only channel",
			input: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL},
			want:  true,
		},
		{
			name:  "Only group",
			input: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
			want:  false,
		},
		{
			name:  "Multiple kinds including channel",
			input: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL, searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
			want:  false,
		},
		{
			name:  "Empty",
			input: []searchv1.SearchChatKind{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOnlyChannelKindFilter(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
