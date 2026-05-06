//go:generate mockgen -destination=mock/search_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/search/transport/grpc SearchUsecase
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/dto/search"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/search/transport/grpc/mock"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestPositiveSearchServer_SearchChats(t *testing.T) {
	type fields struct {
		searchUsecase *mock.MockSearchUsecase
	}

	type args struct {
		ctx context.Context
		req *searchv1.SearchChatsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchv1.SearchChatsResponse
		name    string
		args    args
	}{
		{
			name: "Successful search chats",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "test",
					Kinds:    []searchdomain.ChatType{searchdomain.ChatTypeGroup},
					Limit:    10,
					BeforeID: 0,
				}).Return(&searchdto.SearchChatsResponse{
					Chats: []searchdomain.ChatHit{
						{
							ChatID:             1,
							Type:               "group",
							Title:              "Test Chat",
							AvatarURL:          strPtr("avatar.jpg"),
							LastMessagePreview: strPtr("Hello!"),
							LastMessageAt:      timePtr(now),
							UnreadCount:        2,
						},
					},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "test",
					Kinds:  []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
					Limit:  10,
				},
			},
			want: &searchv1.SearchChatsResponse{
				Chats: []*searchv1.SearchChatItem{
					{
						ChatId:             1,
						Type:               searchv1.ChatType_CHAT_TYPE_GROUP,
						Title:              "Test Chat",
						AvatarUrl:          strPtr("avatar.jpg"),
						LastMessagePreview: strPtr("Hello!"),
						LastMessageAt:      timestamppb.New(now),
						UnreadCount:        2,
					},
				},
				NextBeforeId: 0,
			},
		},
		{
			name: "Search chats with nil response",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), &searchdto.SearchChatsRequest{
					UserID:   200,
					Query:    "nothing",
					Kinds:    nil,
					Limit:    10,
					BeforeID: 0,
				}).Return(nil, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 200,
					Query:  "nothing",
					Limit:  10,
				},
			},
			want: &searchv1.SearchChatsResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchUsecase: mock.NewMockSearchUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SearchServer{
				searchUsecase: f.searchUsecase,
				logger:        zap.NewNop(),
			}

			resp, err := s.SearchChats(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, len(tt.want.GetChats()), len(resp.GetChats()))
			if len(tt.want.GetChats()) > 0 {
				require.Equal(t, tt.want.GetChats()[0].GetChatId(), resp.GetChats()[0].GetChatId())
			}
		})
	}
}

func TestNegativeSearchServer_SearchChats(t *testing.T) {
	type fields struct {
		searchUsecase *mock.MockSearchUsecase
	}

	type args struct {
		ctx context.Context
		req *searchv1.SearchChatsRequest
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Invalid input error",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), &searchdto.SearchChatsRequest{
					UserID:   100,
					Query:    "",
					Kinds:    nil,
					Limit:    10,
					BeforeID: 0,
				}).Return(nil, searchdomain.ErrInvalidInput)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "",
					Limit:  10,
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Not found error",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), gomock.Any()).Return(nil, searchdomain.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Forbidden error",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), gomock.Any()).Return(nil, searchdomain.ErrForbidden)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "Internal error",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), gomock.Any()).Return(nil, searchdomain.ErrInternal)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			wantCode: codes.Internal,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchChats(gomock.Any(), gomock.Any()).Return(nil, errors.New("unknown"))
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchChatsRequest{
					UserId: 100,
					Query:  "test",
					Limit:  10,
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchUsecase: mock.NewMockSearchUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SearchServer{
				searchUsecase: f.searchUsecase,
				logger:        zap.NewNop(),
			}

			_, err := s.SearchChats(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveSearchServer_SearchGlobalChannels(t *testing.T) {
	type fields struct {
		searchUsecase *mock.MockSearchUsecase
	}

	type args struct {
		ctx context.Context
		req *searchv1.SearchGlobalChannelsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchv1.SearchGlobalChannelsResponse
		name    string
		args    args
	}{
		{
			name: "Successful search global channels",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchGlobalChannels(gomock.Any(), &searchdto.SearchGlobalChannelsRequest{
					UserID:   100,
					Query:    "channel",
					Limit:    10,
					BeforeID: 0,
				}).Return(&searchdto.SearchGlobalChannelsResponse{
					Channels: []searchdomain.GlobalChannelHit{
						{
							ChatID:             1,
							Title:              "Channel 1",
							AvatarURL:          strPtr("avatar.jpg"),
							LastMessagePreview: strPtr("Last message"),
							LastMessageAt:      timePtr(now),
							IsMember:           true,
						},
					},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchGlobalChannelsRequest{
					UserId: 100,
					Query:  "channel",
					Limit:  10,
				},
			},
			want: &searchv1.SearchGlobalChannelsResponse{
				Channels: []*searchv1.SearchGlobalChannelItem{
					{
						ChatId:             1,
						Title:              "Channel 1",
						AvatarUrl:          strPtr("avatar.jpg"),
						LastMessagePreview: strPtr("Last message"),
						LastMessageAt:      timestamppb.New(now),
						IsMember:           true,
					},
				},
				NextBeforeId: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchUsecase: mock.NewMockSearchUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SearchServer{
				searchUsecase: f.searchUsecase,
				logger:        zap.NewNop(),
			}

			resp, err := s.SearchGlobalChannels(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, len(tt.want.GetChannels()), len(resp.GetChannels()))
		})
	}
}

func TestPositiveSearchServer_SearchContacts(t *testing.T) {
	type fields struct {
		searchUsecase *mock.MockSearchUsecase
	}

	type args struct {
		ctx context.Context
		req *searchv1.SearchContactsRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchv1.SearchContactsResponse
		name    string
		args    args
	}{
		{
			name: "Successful search contacts",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchContacts(gomock.Any(), &searchdto.SearchContactsRequest{
					UserID:   100,
					Query:    "john",
					Scope:    searchdomain.ContactScopeContacts,
					Limit:    10,
					BeforeID: 0,
				}).Return(&searchdto.SearchContactsResponse{
					Contacts: []searchdomain.ContactHit{
						{
							UserID:      101,
							DisplayName: "John Doe",
							Login:       strPtr("john_doe"),
							AvatarURL:   strPtr("avatar.jpg"),
							IsOnline:    true,
							LastSeenAt:  timePtr(now),
						},
					},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchContactsRequest{
					UserId: 100,
					Query:  "john",
					Scope:  searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS,
					Limit:  10,
				},
			},
			want: &searchv1.SearchContactsResponse{
				Contacts: []*searchv1.SearchContactItem{
					{
						UserId:      101,
						DisplayName: "John Doe",
						Login:       strPtr("john_doe"),
						AvatarUrl:   strPtr("avatar.jpg"),
						IsOnline:    true,
						LastSeenAt:  timestamppb.New(now),
					},
				},
				NextBeforeId: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchUsecase: mock.NewMockSearchUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SearchServer{
				searchUsecase: f.searchUsecase,
				logger:        zap.NewNop(),
			}

			resp, err := s.SearchContacts(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, len(tt.want.GetContacts()), len(resp.GetContacts()))
		})
	}
}

func TestPositiveSearchServer_SearchMessagesInChat(t *testing.T) {
	type fields struct {
		searchUsecase *mock.MockSearchUsecase
	}

	type args struct {
		ctx context.Context
		req *searchv1.SearchMessagesInChatRequest
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *searchv1.SearchMessagesInChatResponse
		name    string
		args    args
	}{
		{
			name: "Successful search messages in chat",
			prepare: func(f *fields) {
				f.searchUsecase.EXPECT().SearchMessagesInChat(gomock.Any(), &searchdto.SearchMessagesInChatRequest{
					UserID:   100,
					ChatID:   50,
					Query:    "hello",
					Limit:    10,
					BeforeID: 0,
				}).Return(&searchdto.SearchMessagesInChatResponse{
					Messages: []searchdomain.MessageHit{
						{
							MessageID:   1000,
							ChatID:      50,
							SenderID:    101,
							TextPreview: "hello world",
							CreatedAt:   now,
							Highlights: []searchdomain.MessageHighlight{
								{Fragment: "hello"},
							},
						},
					},
					NextBeforeID: 0,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &searchv1.SearchMessagesInChatRequest{
					UserId: 100,
					ChatId: 50,
					Query:  "hello",
					Limit:  10,
				},
			},
			want: &searchv1.SearchMessagesInChatResponse{
				Messages: []*searchv1.SearchMessageItem{
					{
						MessageId:   1000,
						ChatId:      50,
						SenderId:    101,
						TextPreview: "hello world",
						CreatedAt:   timestamppb.New(now),
						Highlights: &searchv1.SearchMessageHighlight{
							Fragment: "hello",
						},
					},
				},
				NextBeforeId: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				searchUsecase: mock.NewMockSearchUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SearchServer{
				searchUsecase: f.searchUsecase,
				logger:        zap.NewNop(),
			}

			resp, err := s.SearchMessagesInChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, len(tt.want.GetMessages()), len(resp.GetMessages()))
		})
	}
}

func TestMapDomainErrToProtoErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "Nil error",
			err:      nil,
			wantCode: codes.OK,
		},
		{
			name:     "Invalid input",
			err:      searchdomain.ErrInvalidInput,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "Not found",
			err:      searchdomain.ErrNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "Forbidden",
			err:      searchdomain.ErrForbidden,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "Internal",
			err:      searchdomain.ErrInternal,
			wantCode: codes.Internal,
		},
		{
			name:     "Unknown error",
			err:      errors.New("some error"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapDomainErrToProtoErr(tt.err)
			if tt.err == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestMapSearchChatKindsProtoToDomain(t *testing.T) {
	tests := []struct {
		name  string
		kinds []searchv1.SearchChatKind
		want  []searchdomain.ChatType
	}{
		{
			name:  "Empty kinds",
			kinds: []searchv1.SearchChatKind{},
			want:  nil,
		},
		{
			name:  "Dialog kind",
			kinds: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG},
			want:  []searchdomain.ChatType{searchdomain.ChatTypeDialog},
		},
		{
			name:  "Group kind",
			kinds: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP},
			want:  []searchdomain.ChatType{searchdomain.ChatTypeGroup},
		},
		{
			name:  "Channel kind",
			kinds: []searchv1.SearchChatKind{searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL},
			want:  []searchdomain.ChatType{searchdomain.ChatTypeChannel},
		},
		{
			name:  "Multiple kinds",
			kinds: []searchv1.SearchChatKind{
				searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG,
				searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP,
			},
			want: []searchdomain.ChatType{searchdomain.ChatTypeDialog, searchdomain.ChatTypeGroup},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSearchChatKindsProtoToDomain(tt.kinds)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMapContactScopeProtoToDomain(t *testing.T) {
	tests := []struct {
		name  string
		scope searchv1.SearchContactScope
		want  searchdomain.ContactScope
	}{
		{
			name:  "Contacts scope",
			scope: searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS,
			want:  searchdomain.ContactScopeContacts,
		},
		{
			name:  "Local scope",
			scope: searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL,
			want:  searchdomain.ContactScopeLocal,
		},
		{
			name:  "Unspecified scope",
			scope: searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_UNSPECIFIED,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapContactScopeProtoToDomain(tt.scope)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMapChatTypeDomainToProto(t *testing.T) {
	tests := []struct {
		name string
		typ  searchdomain.ChatType
		want searchv1.ChatType
	}{
		{
			name: "Dialog type",
			typ:  searchdomain.ChatTypeDialog,
			want: searchv1.ChatType_CHAT_TYPE_DIALOG,
		},
		{
			name: "Group type",
			typ:  searchdomain.ChatTypeGroup,
			want: searchv1.ChatType_CHAT_TYPE_GROUP,
		},
		{
			name: "Channel type",
			typ:  searchdomain.ChatTypeChannel,
			want: searchv1.ChatType_CHAT_TYPE_CHANNEL,
		},
		{
			name: "Unknown type",
			typ:  "unknown",
			want: searchv1.ChatType_CHAT_TYPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapChatTypeDomainToProto(tt.typ)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMapMessageHighlightsDomainToProto(t *testing.T) {
	tests := []struct {
		name       string
		highlights []searchdomain.MessageHighlight
		want       *searchv1.SearchMessageHighlight
	}{
		{
			name:       "Empty highlights",
			highlights: []searchdomain.MessageHighlight{},
			want:       nil,
		},
		{
			name: "Highlight with fragment",
			highlights: []searchdomain.MessageHighlight{
				{Fragment: "hello"},
			},
			want: &searchv1.SearchMessageHighlight{Fragment: "hello"},
		},
		{
			name: "Multiple highlights - returns first non-empty",
			highlights: []searchdomain.MessageHighlight{
				{Fragment: ""},
				{Fragment: "world"},
			},
			want: &searchv1.SearchMessageHighlight{Fragment: "world"},
		},
		{
			name: "All empty fragments",
			highlights: []searchdomain.MessageHighlight{
				{Fragment: ""},
				{Fragment: ""},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapMessageHighlightsDomainToProto(tt.highlights)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want.GetFragment(), got.GetFragment())
			}
		})
	}
}

func TestMapSearchGlobalChannelsRequestProtoToDTO_Nil(t *testing.T) {
	result := mapSearchGlobalChannelsRequestProtoToDTO(nil)
	require.Nil(t, result)
}

func TestMapSearchChatsRequestProtoToDTO_Nil(t *testing.T) {
	result := mapSearchChatsRequestProtoToDTO(nil)
	require.Nil(t, result)
}

func TestMapSearchContactsRequestProtoToDTO_Nil(t *testing.T) {
	result := mapSearchContactsRequestProtoToDTO(nil)
	require.Nil(t, result)
}

func TestMapSearchMessagesInChatRequestProtoToDTO_Nil(t *testing.T) {
	result := mapSearchMessagesInChatRequestProtoToDTO(nil)
	require.Nil(t, result)
}

func TestMapGlobalChannelHitDomainToProto_Nil(t *testing.T) {
	result := mapGlobalChannelHitDomainToProto(nil)
	require.Nil(t, result)
}

func TestMapChatHitDomainToProto_Nil(t *testing.T) {
	result := mapChatHitDomainToProto(nil)
	require.Nil(t, result)
}

func TestMapContactHitDomainToProto_Nil(t *testing.T) {
	result := mapContactHitDomainToProto(nil)
	require.Nil(t, result)
}

func TestMapMessageHitDomainToProto_Nil(t *testing.T) {
	result := mapMessageHitDomainToProto(nil)
	require.Nil(t, result)
}