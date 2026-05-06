//go:generate mockgen -destination=mock/chat_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1 ChatClient
package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"mime/multipart" 
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/chat/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveGatewayChatHandler_GetChats(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		userID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful get chats",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChats(gomock.Any(), &chatv1.RequestGetUserChats{
					UserId: 100,
				}).Return(&chatv1.ResponseGetUserChats{
					Chats: []*chatv1.ChatInformation{
						{
							Id:    1,
							Type:  chatv1.ChatType_GROUP,
							Title: "Group Chat",
							LastMessage: &chatv1.MessageInformation{
								SenderId:  101,
								Text:      "Hello!",
								CreatedAt: timestamppb.Now(),
							},
							Avatar:  strPtr("avatar.jpg"),
							OwnerId: 100,
						},
						{
							Id:    2,
							Type:  chatv1.ChatType_DIALOG,
							Title: "Dialog",
							LastMessage: &chatv1.MessageInformation{
								SenderId:  102,
								Text:      "Hi!",
								CreatedAt: timestamppb.Now(),
							},
							Avatar:  nil,
							OwnerId: 100,
						},
					},
				}, nil)
			},
			args: args{userID: 100},
			want: http.StatusOK,
		},
		{
			name: "Get chats empty list",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChats(gomock.Any(), &chatv1.RequestGetUserChats{
					UserId: 200,
				}).Return(&chatv1.ResponseGetUserChats{
					Chats: []*chatv1.ChatInformation{},
				}, nil)
			},
			args: args{userID: 200},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Get("/chats", handler.GetChats)

			req := httptest.NewRequest(http.MethodGet, "/chats", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_GetChats(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		want       int
		userID     interface{}
	}{
		{
			name:       "Missing user_id in context",
			prepare:    nil,
			want:       http.StatusUnauthorized,
			userID:     nil,
		},
		{
			name:       "Invalid user_id type",
			prepare:    nil,
			want:       http.StatusUnauthorized,
			userID:     "invalid",
		},
		{
			name: "Chat service error",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChats(gomock.Any(), &chatv1.RequestGetUserChats{
					UserId: 100,
				}).Return(nil, grpcerr.New(codes.Internal, int32(chatv1.ChatErrorCode_CHAT_ERROR_INTERNAL), "internal error"))
			},
			want:   http.StatusInternalServerError,
			userID: int64(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Get("/chats", handler.GetChats)

			req := httptest.NewRequest(http.MethodGet, "/chats", nil)
			if tt.userID != nil {
				if uid, ok := tt.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				} else {
					ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
					req = req.WithContext(ctx)
				}
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_CreateChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		body map[string]interface{}
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful create group chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), &chatv1.RequestChatCreate{
					UserId:    100,
					Type:      chatv1.ChatType_GROUP,
					Title:     "New Group",
					MembersId: []int64{101, 102},
				}).Return(&chatv1.ChatInformation{
					Id:      10,
					Type:    chatv1.ChatType_GROUP,
					Title:   "New Group",
					OwnerId: 100,
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type":       "group",
					"title":      "New Group",
					"members_id": []int64{101, 102},
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful create dialog chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), &chatv1.RequestChatCreate{
					UserId:    100,
					Type:      chatv1.ChatType_DIALOG,
					Title:     "",
					MembersId: []int64{101},
				}).Return(&chatv1.ChatInformation{
					Id:      20,
					Type:    chatv1.ChatType_DIALOG,
					Title:   "Friend",
					OwnerId: 100,
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type":       "dialog",
					"members_id": []int64{101},
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful create channel chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), &chatv1.RequestChatCreate{
					UserId:    100,
					Type:      chatv1.ChatType_CHANNEL,
					Title:     "New Channel",
					MembersId: nil,
				}).Return(&chatv1.ChatInformation{
					Id:      30,
					Type:    chatv1.ChatType_CHANNEL,
					Title:   "New Channel",
					OwnerId: 100,
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type":  "channel",
					"title": "New Channel",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats", handler.CreateChat)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_CreateChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		body interface{}
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Invalid JSON body",
			args: args{
				body: "invalid json",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing type field",
			args: args{
				body: map[string]interface{}{
					"title": "New Group",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid chat type",
			args: args{
				body: map[string]interface{}{
					"type":  "invalid",
					"title": "New Group",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Group without title",
			args: args{
				body: map[string]interface{}{
					"type":       "group",
					"members_id": []int64{101},
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Chat not found error",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.NotFound, int32(chatv1.ChatErrorCode_CHAT_ERROR_CHAT_NOT_FOUND), "chat not found"))
			},
			args: args{
				body: map[string]interface{}{
					"type":  "group",
					"title": "New Group",
				},
			},
			want: http.StatusNotFound,
		},
		{
			name: "Dialog already exists",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.AlreadyExists, int32(chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_EXISTS), "dialog already exists"))
			},
			args: args{
				body: map[string]interface{}{
					"type":       "dialog",
					"members_id": []int64{101},
				},
			},
			want: http.StatusConflict,
		},
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.NotFound, int32(chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_FOUND), "user not found"))
			},
			args: args{
				body: map[string]interface{}{
					"type":       "dialog",
					"members_id": []int64{999},
				},
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats", handler.CreateChat)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_GetChatByID(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful get chat by id",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChatByID(gomock.Any(), &chatv1.RequestGetChatByID{
					UserId: 100,
					ChatId: 1,
				}).Return(&chatv1.ChatInformation{
					Id:      1,
					Type:    chatv1.ChatType_GROUP,
					Title:   "Group Chat",
					OwnerId: 100,
				}, nil)
			},
			args: args{chatID: 1},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Get("/chats/{id}", handler.GetChatByID)

			req := httptest.NewRequest(http.MethodGet, "/chats/1", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_GetChatByID(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		want       int
		chatID     string
		userID     interface{}
	}{
		{
			name:       "Invalid chat id",
			want:       http.StatusBadRequest,
			chatID:     "invalid",
			userID:     int64(100),
		},
		{
			name:       "Missing user_id",
			want:       http.StatusUnauthorized,
			chatID:     "1",
			userID:     nil,
		},
		{
			name: "Chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChatByID(gomock.Any(), &chatv1.RequestGetChatByID{
					UserId: 100,
					ChatId: 999,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(chatv1.ChatErrorCode_CHAT_ERROR_CHAT_NOT_FOUND), "chat not found"))
			},
			want:   http.StatusNotFound,
			chatID: "999",
			userID: int64(100),
		},
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChatByID(gomock.Any(), &chatv1.RequestGetChatByID{
					UserId: 100,
					ChatId: 1,
				}).Return(nil, grpcerr.New(codes.PermissionDenied, int32(chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_MEMBER), "user not member"))
			},
			want:   http.StatusForbidden,
			chatID: "1",
			userID: int64(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Get("/chats/{id}", handler.GetChatByID)

			req := httptest.NewRequest(http.MethodGet, "/chats/"+tt.chatID, nil)
			if tt.userID != nil {
				if uid, ok := tt.userID.(int64); ok {
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

func TestPositiveGatewayChatHandler_DeleteChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		chatID  string
	}{
		{
			name: "Successful delete chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().DeleteChat(gomock.Any(), &chatv1.RequestDeleteChat{
					UserId: 100,
					ChatId: 1,
				}).Return(&emptypb.Empty{}, nil)
			},
			chatID: "1",
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Delete("/chats/{id}", handler.DeleteChat)

			req := httptest.NewRequest(http.MethodDelete, "/chats/"+tt.chatID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID string
		body   map[string]string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful update chat title",
			prepare: func(f *fields) {
				f.chatService.EXPECT().UpdateChatTitle(gomock.Any(), &chatv1.RequestUpdateTitle{
					UserId: 100,
					ChatId: 1,
					Title:  "New Title",
				}).Return(&chatv1.ChatInformation{
					Id:    1,
					Title: "New Title",
				}, nil)
			},
			args: args{
				chatID: "1",
				body: map[string]string{
					"title": "New Title",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Put("/chats/{id}/title", handler.UpdateChatTitle)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/chats/"+tt.args.chatID+"/title", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_AddMembers(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID string
		body   map[string]interface{}
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful add members",
			prepare: func(f *fields) {
				f.chatService.EXPECT().AddMembersToChat(gomock.Any(), &chatv1.RequestAddMembersToChat{
					UserId:    100,
					ChatId:    1,
					MembersId: []int64{101, 102},
				}).Return(&emptypb.Empty{}, nil)
			},
			args: args{
				chatID: "1",
				body: map[string]interface{}{
					"members_id": []int64{101, 102},
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats/{id}/members", handler.AddMembers)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/chats/"+tt.args.chatID+"/members", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_DeleteMember(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID   string
		memberID string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful delete member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().DeleteMemberFromChat(gomock.Any(), &chatv1.RequestDeleteMemberFromChat{
					UserId:   100,
					ChatId:   1,
					MemberId: 101,
				}).Return(&emptypb.Empty{}, nil)
			},
			args: args{
				chatID:   "1",
				memberID: "101",
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Delete("/chats/{id}/members/{member_id}", handler.DeleteMember)

			req := httptest.NewRequest(http.MethodDelete, "/chats/"+tt.args.chatID+"/members/"+tt.args.memberID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_GetChatMembers(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		chatID  string
	}{
		{
			name: "Successful get chat members",
			prepare: func(f *fields) {
				f.chatService.EXPECT().GetChatMembers(gomock.Any(), &chatv1.RequestChatMembers{
					UserId: 100,
					ChatId: 1,
				}).Return(&chatv1.ResponseGetChatMembers{
					MembersId: []int64{100, 101, 102},
				}, nil)
			},
			chatID: "1",
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Get("/chats/{id}/members", handler.GetChatMembers)

			req := httptest.NewRequest(http.MethodGet, "/chats/"+tt.chatID+"/members", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayChatHandler_QuitChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		chatID  string
	}{
		{
			name: "Successful quit chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().QuitChat(gomock.Any(), &chatv1.RequestQuitChat{
					UserId: 100,
					ChatId: 1,
				}).Return(&emptypb.Empty{}, nil)
			},
			chatID: "1",
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats/{id}/quit", handler.QuitChat)

			req := httptest.NewRequest(http.MethodPost, "/chats/"+tt.chatID+"/quit", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID   string
		userID   interface{}
		fileData []byte
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
				chatID:   "1",
				userID:   nil,
				fileData: []byte("data"),
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid chat id",
			args: args{
				chatID:   "invalid",
				userID:   int64(100),
				fileData: []byte("data"),
			},
			want: http.StatusBadRequest,
		},
		{
			name: "No file uploaded",
			args: args{
				chatID:   "1",
				userID:   int64(100),
				fileData: nil,
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().UpdateChatAvatar(gomock.Any(), gomock.Any()).Return(nil,
					grpcerr.New(codes.NotFound, int32(chatv1.ChatErrorCode_CHAT_ERROR_CHAT_NOT_FOUND), "chat not found"))
			},
			args: args{
				chatID:   "1",
				userID:   int64(100),
				fileData: []byte("data"),
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Put("/chats/{id}/avatar", handler.UpdateChatAvatar)

			var req *http.Request
			if tt.args.fileData != nil {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("avatar", "avatar.png")
				part.Write(tt.args.fileData)
				writer.Close()
				req = httptest.NewRequest(http.MethodPut, "/chats/"+tt.args.chatID+"/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req = httptest.NewRequest(http.MethodPut, "/chats/"+tt.args.chatID+"/avatar", nil)
			}

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

func TestPositiveGatewayChatHandler_UpdateChatDescription(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID      string
		body        map[string]string
		userID      int64
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful update chat description",
			prepare: func(f *fields) {
				f.chatService.EXPECT().UpdateChatDescription(gomock.Any(), &chatv1.RequestUpdateDescription{
					UserId:      100,
					ChatId:      1,
					Description: "New description",
				}).Return(&chatv1.ChatInformation{
					Id:          1,
					Type:        chatv1.ChatType_GROUP,
					Title:       "Group Chat",
					Description: strPtr("New description"),
				}, nil)
			},
			args: args{
				chatID: "1",
				userID: 100,
				body: map[string]string{
					"description": "New description",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Put("/chats/{id}/description", handler.UpdateChatDescription)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/chats/"+tt.args.chatID+"/description", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_UpdateChatDescription(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	type args struct {
		chatID string
		body   interface{}
		userID interface{}
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
				chatID: "1",
				userID: nil,
				body:   map[string]string{"description": "test"},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid chat id",
			args: args{
				chatID: "invalid",
				userID: int64(100),
				body:   map[string]string{"description": "test"},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid JSON body",
			args: args{
				chatID: "1",
				userID: int64(100),
				body:   "invalid json",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Dialog cannot have description",
			prepare: func(f *fields) {
				f.chatService.EXPECT().UpdateChatDescription(gomock.Any(), gomock.Any()).Return(nil,
					grpcerr.New(codes.FailedPrecondition, int32(chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_DESCRIPTION), "dialog can't have description"))
			},
			args: args{
				chatID: "1",
				userID: int64(100),
				body:   map[string]string{"description": "test"},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Only owner can change description",
			prepare: func(f *fields) {
				f.chatService.EXPECT().UpdateChatDescription(gomock.Any(), gomock.Any()).Return(nil,
					grpcerr.New(codes.PermissionDenied, int32(chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_CHANGE_DESCRIPTION), "only owner can change description"))
			},
			args: args{
				chatID: "1",
				userID: int64(100),
				body:   map[string]string{"description": "test"},
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Put("/chats/{id}/description", handler.UpdateChatDescription)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/chats/"+tt.args.chatID+"/description", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

func TestPositiveGatewayChatHandler_JoinChannel(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		chatID  string
		userID  int64
	}{
		{
			name:   "Successful join channel",
			userID: 100,
			chatID: "1",
			prepare: func(f *fields) {
				f.chatService.EXPECT().JoinChannel(gomock.Any(), &chatv1.RequestJoinChannel{
					UserId: 100,
					ChatId: 1,
				}).Return(&emptypb.Empty{}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats/{id}/join", handler.JoinChannel)

			req := httptest.NewRequest(http.MethodPost, "/chats/"+tt.chatID+"/join", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayChatHandler_JoinChannel(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		chatID  string
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			userID: nil,
			chatID: "1",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "Invalid chat id",
			userID: int64(100),
			chatID: "invalid",
			want:   http.StatusBadRequest,
		},
		{
			name:   "Only channel can be joined",
			userID: int64(100),
			chatID: "1",
			prepare: func(f *fields) {
				f.chatService.EXPECT().JoinChannel(gomock.Any(), &chatv1.RequestJoinChannel{
					UserId: 100,
					ChatId: 1,
				}).Return(nil, grpcerr.New(codes.FailedPrecondition, int32(chatv1.ChatErrorCode_CHAT_ERROR_ONLY_CHANNEL_CAN_BE_JOINED), "only channel can be joined"))
			},
			want: http.StatusBadRequest,
		},
		{
			name:   "Chat not found",
			userID: int64(100),
			chatID: "999",
			prepare: func(f *fields) {
				f.chatService.EXPECT().JoinChannel(gomock.Any(), &chatv1.RequestJoinChannel{
					UserId: 100,
					ChatId: 999,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(chatv1.ChatErrorCode_CHAT_ERROR_CHAT_NOT_FOUND), "chat not found"))
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayChatHandler{
				ChatService: f.chatService,
			}

			r := chi.NewRouter()
			r.Post("/chats/{id}/join", handler.JoinChannel)

			req := httptest.NewRequest(http.MethodPost, "/chats/"+tt.chatID+"/join", nil)
			if tt.userID != nil {
				if uid, ok := tt.userID.(int64); ok {
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

func TestGatewayChatHandler_MapChatType(t *testing.T) {
	tests := []struct {
		name string
		typ  chatv1.ChatType
		want string
	}{
		{
			name: "Group type",
			typ:  chatv1.ChatType_GROUP,
			want: "group",
		},
		{
			name: "Channel type",
			typ:  chatv1.ChatType_CHANNEL,
			want: "channel",
		},
		{
			name: "Dialog type",
			typ:  chatv1.ChatType_DIALOG,
			want: "dialog",
		},
		{
			name: "Unspecified type defaults to dialog",
			typ:  chatv1.ChatType(999),
			want: "dialog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapChatType(tt.typ)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGatewayChatHandler_ParseChatType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    chatv1.ChatType
		wantErr bool
	}{
		{
			name:    "Group type",
			input:   "group",
			want:    chatv1.ChatType_GROUP,
			wantErr: false,
		},
		{
			name:    "Group type uppercase",
			input:   "GROUP",
			want:    chatv1.ChatType_GROUP,
			wantErr: false,
		},
		{
			name:    "Channel type",
			input:   "channel",
			want:    chatv1.ChatType_CHANNEL,
			wantErr: false,
		},
		{
			name:    "Dialog type",
			input:   "dialog",
			want:    chatv1.ChatType_DIALOG,
			wantErr: false,
		},
		{
			name:    "Invalid type",
			input:   "invalid",
			want:    chatv1.ChatType_DIALOG,
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    chatv1.ChatType_DIALOG,
			wantErr: true,
		},
		{
			name:    "With spaces",
			input:   "  group  ",
			want:    chatv1.ChatType_GROUP,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChatType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGatewayChatHandler_MapChatInfo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		c    *chatv1.ChatInformation
		want ChatInfoResponse
	}{
		{
			name: "Full chat info",
			c: &chatv1.ChatInformation{
				Id:      1,
				Type:    chatv1.ChatType_GROUP,
				Title:   "Group Chat",
				OwnerId: 100,
				Avatar:  strPtr("avatar.jpg"),
				Description: strPtr("Description"),
				LastMessage: &chatv1.MessageInformation{
					SenderId:  101,
					Text:      "Hello!",
					CreatedAt: timestamppb.New(now),
				},
			},
			want: ChatInfoResponse{
				ID:      1,
				Type:    "group",
				Title:   "Group Chat",
				OwnerID: 100,
				Avatar:  strPtr("avatar.jpg"),
				Description: strPtr("Description"),
				LastMessage: MessageInfoResponse{
					SenderID:  101,
					Text:      "Hello!",
					CreatedAt: now,
				},
			},
		},
		{
			name: "Chat without optional fields",
			c: &chatv1.ChatInformation{
				Id:      2,
				Type:    chatv1.ChatType_DIALOG,
				Title:   "Dialog",
				OwnerId: 200,
			},
			want: ChatInfoResponse{
				ID:      2,
				Type:    "dialog",
				Title:   "Dialog",
				OwnerID: 200,
				LastMessage: MessageInfoResponse{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapChatInfo(tt.c)
			require.Equal(t, tt.want.ID, got.ID)
			require.Equal(t, tt.want.Type, got.Type)
			require.Equal(t, tt.want.Title, got.Title)
			require.Equal(t, tt.want.OwnerID, got.OwnerID)
			if tt.want.Avatar != nil {
				require.Equal(t, *tt.want.Avatar, *got.Avatar)
			}
			if tt.want.Description != nil {
				require.Equal(t, *tt.want.Description, *got.Description)
			}
		})
	}
}

func TestGatewayChatHandler_UserID(t *testing.T) {
	tests := []struct {
		name     string
		ctxValue interface{}
		wantID   int64
		wantOK   bool
	}{
		{
			name:     "Valid user_id",
			ctxValue: int64(100),
			wantID:   100,
			wantOK:   true,
		},
		{
			name:     "Missing user_id",
			ctxValue: nil,
			wantID:   0,
			wantOK:   false,
		},
		{
			name:     "Invalid type",
			ctxValue: "string",
			wantID:   0,
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.ctxValue != nil {
				ctx := context.WithValue(req.Context(), middleware.UserID, tt.ctxValue)
				req = req.WithContext(ctx)
			}
			gotID, gotOK := userID(req)
			require.Equal(t, tt.wantID, gotID)
			require.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func TestGatewayChatHandler_ChatIDParam(t *testing.T) {
	tests := []struct {
		name       string
		urlParam   string
		wantID     int64
		wantErr    bool
	}{
		{
			name:     "Valid id",
			urlParam: "123",
			wantID:   123,
			wantErr:  false,
		},
		{
			name:     "Invalid id",
			urlParam: "invalid",
			wantID:   0,
			wantErr:  true,
		},
		{
			name:     "Empty id",
			urlParam: "",
			wantID:   0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/chats/{id}", func(w http.ResponseWriter, r *http.Request) {})
			req := httptest.NewRequest(http.MethodGet, "/chats/"+tt.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.urlParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			gotID, err := chatIDParam(req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestGatewayChatHandler_SendUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	sendUnauthorized(w)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGatewayChatHandler_SendInvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	sendInvalidID(w)
	require.Equal(t, http.StatusBadRequest, w.Code)
}