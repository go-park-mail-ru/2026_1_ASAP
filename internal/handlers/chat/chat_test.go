package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/chat/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
)

func decodeAPIErrorChat(t *testing.T, rr *httptest.ResponseRecorder) dtoApi.ApiErrorResponse {
	t.Helper()
	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	return got
}

func TestPositiveChatHandler_GetChats(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		want    []dto.ChatInformationDTO
		args    args
	}{
		{
			name: "success get chats",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetAllChats(gomock.Any(), int64(100)).
					Return([]dto.ChatInformationDTO{
						{
							ID:       1,
							ChatType: dto.ChatTypeGroup,
							Title:    "Group Chat",
							LastMessage: dto.MessageDTO{
								Text:      "Hello",
								CreatedAt: time.Now(),
							},
							Avatar: strPtrChat("avatar.jpg"),
						},
					}, nil)
			},
			args: args{userID: 100},
			want: []dto.ChatInformationDTO{
				{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "Group Chat",
					LastMessage: dto.MessageDTO{
						Text:      "Hello",
						CreatedAt: time.Now(),
					},
					Avatar: strPtrChat("avatar.jpg"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			h.GetChats(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[[]dto.ChatInformationDTO]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, tt.want[0].ID, resp.Body[0].ID)
			require.Equal(t, tt.want[0].ChatType, resp.Body[0].ChatType)
			require.Equal(t, tt.want[0].Title, resp.Body[0].Title)
			require.Equal(t, tt.want[0].Avatar, resp.Body[0].Avatar)
			require.Equal(t, tt.want[0].LastMessage.Text, resp.Body[0].LastMessage.Text)
		})
	}
}

func TestNegativeChatHandler_GetChats(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
	}

	tests := []struct {
		args       args
		prepare    func(*fields)
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetAllChats(gomock.Any(), int64(100)).
					Return(nil, errors.New("db error"))
			},
			args:     args{userID: int64(100)},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			h.GetChats(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIErrorChat(t, rr)
			require.NotEmpty(t, got.Errors)
			if !tt.skipErrCmp {
				require.Equal(t, tt.wantErr, got.Errors[0].Code)
			}
		})
	}
}

func TestPositiveChatHandler_ChatCreate(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		chatType dto.ChatType
		title    string
		members  []int64
		userID   int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "success create group chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					CreateChat(gomock.Any(), gomock.Any(), int64(100)).
					Return(&dto.ChatInformationDTO{
						ID:       10,
						ChatType: dto.ChatTypeGroup,
						Title:    "New Group",
					}, nil)

				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(10)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(10), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       10,
						ChatType: dto.ChatTypeGroup,
						Title:    "New Group",
					}, nil).AnyTimes()

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(2)
			},
			args: args{
				userID:   100,
				chatType: dto.ChatTypeGroup,
				title:    "New Group",
				members:  []int64{101},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			reqBody := dto.ChatCreate{
				Type:      tt.args.chatType,
				Title:     tt.args.title,
				MembersID: tt.args.members,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			h.ChatCreate(rr, req)

			require.Equal(t, http.StatusCreated, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, int64(10), resp.Body.ID)
		})
	}
}

func TestNegativeChatHandler_ChatCreate(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		body   string
	}

	tests := []struct {
		prepare    func(*fields)
		name       string
		wantErr    dtoApi.ErrorCode
		args       args
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, body: `{}`},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:       "invalid json",
			prepare:    nil,
			args:       args{userID: int64(100), body: `{invalid`},
			wantCode:   http.StatusBadRequest,
			wantErr:    dtoApi.InvalidJson,
			skipErrCmp: false,
		},
		{
			name:       "validation fails - empty title for group",
			prepare:    nil,
			args:       args{userID: int64(100), body: `{"type":"group","title":"","members_id":[]}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "dialog already exists",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					CreateChat(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, domain.ErrDialogAlreadyExists)
			},
			args: args{
				userID: int64(100),
				body:   `{"type":"dialog","title":"","members_id":[100,101]}`,
			},
			wantCode: http.StatusConflict,
			wantErr:  dtoApi.DialogAlreadyExists,
		},
		{
			name: "user not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					CreateChat(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, domainUser.ErrNotFound)
			},
			args: args{
				userID: int64(100),
				body:   `{"type":"group","title":"New Group","members_id":[101]}`,
			},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.UserNotFound,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					CreateChat(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, errors.New("db error"))
			},
			args: args{
				userID: int64(100),
				body:   `{"type":"group","title":"New Group","members_id":[101]}`,
			},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.CreateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			h.ChatCreate(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_GetChatByID(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID int64
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "success get chat by id",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), int64(100)).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
						LastMessage: dto.MessageDTO{
							Text:      "Hello",
							CreatedAt: time.Now(),
						},
						Avatar: strPtrChat("avatar.jpg"),
					}, nil)
			},
			args: args{userID: 100, chatID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10), nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.GetChatByID(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, tt.args.chatID, resp.Body.ID)
			require.Equal(t, "Group Chat", resp.Body.Title)
		})
	}
}

func TestNegativeChatHandler_GetChatByID(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
		path   string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1", path: "/api/v1/chats/1"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid", path: "/api/v1/chats/invalid"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), int64(100)).
					Return(nil, domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1", path: "/api/v1/chats/1"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.AccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.args.path, nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.GetChatByID(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_DeleteChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID int64
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "success delete chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					DeleteChat(gomock.Any(), int64(100), int64(1)).
					Return(nil)

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(2)
			},
			args: args{userID: 100, chatID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10), nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteChat(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[any]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
		})
	}
}

func TestNegativeChatHandler_DeleteChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "only owner can delete",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)
				f.chatService.EXPECT().
					DeleteChat(gomock.Any(), int64(101), int64(1)).
					Return(domain.ErrOnlyOwnerCanDeleteChat)
			},
			args:     args{userID: int64(101), chatID: "1"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.CantDeleteChat,
		},
		{
			name: "chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)
				f.chatService.EXPECT().
					DeleteChat(gomock.Any(), int64(100), int64(1)).
					Return(domain.ErrChatNotFound)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.CantFindChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+tt.args.chatID, nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteChat(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		fileName    string
		fileContent string
		contentType string
		userID      int64
		chatID      int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "success update avatar",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatAvatar(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
						LastMessage: dto.MessageDTO{
							Text:      "Hello",
							CreatedAt: time.Now(),
						},
						Avatar: strPtrChat("new_avatar.jpg"),
					}, nil)

				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
						LastMessage: dto.MessageDTO{
							Text:      "Hello",
							CreatedAt: time.Now(),
						},
						Avatar: strPtrChat("new_avatar.jpg"),
					}, nil).AnyTimes()

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(2)
			},
			args: args{
				userID:      100,
				chatID:      1,
				fileName:    "avatar.jpg",
				fileContent: "fake-image-content",
				contentType: "image/jpeg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("avatar", tt.args.fileName)
			part.Write([]byte(tt.args.fileContent))
			writer.Close()

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/avatar", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.UpdateChatAvatar(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, int64(1), resp.Body.ID)
			require.Equal(t, "new_avatar.jpg", *resp.Body.Avatar)
		})
	}
}

func TestNegativeChatHandler_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID      interface{}
		chatID      string
		fileName    string
		fileContent string
		contentType string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1", fileName: "avatar.jpg", fileContent: "test", contentType: "image/jpeg"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid", fileName: "avatar.jpg", fileContent: "test", contentType: "image/jpeg"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name:     "no file",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "1", fileName: "", fileContent: "", contentType: ""},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.FileNotFound,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatAvatar(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil, domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1", fileName: "avatar.jpg", fileContent: "test", contentType: "image/jpeg"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
		{
			name: "dialog cannot have avatar",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatAvatar(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil, domain.ErrDialogCannotHaveCustomAvatar)
			},
			args:     args{userID: int64(100), chatID: "1", fileName: "avatar.jpg", fileContent: "test", contentType: "image/jpeg"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.CantChangeAvatar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			var req *http.Request

			if tt.args.fileName != "" {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("avatar", tt.args.fileName)
				part.Write([]byte(tt.args.fileContent))
				writer.Close()
				req = httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+tt.args.chatID+"/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req = httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+tt.args.chatID+"/avatar", nil)
			}

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.UpdateChatAvatar(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_GetChatMembers(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID int64
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseGetChatMembers
		name    string
		args    args
	}{
		{
			name: "success get chat members",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetAllChatMembers(gomock.Any(), int64(100), int64(1)).
					Return(&dto.ResponseGetChatMembers{
						MembersId: []int64{100, 101, 102},
					}, nil)
			},
			args: args{userID: 100, chatID: 1},
			want: &dto.ResponseGetChatMembers{
				MembersId: []int64{100, 101, 102},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/members", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.GetChatMembers(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ResponseGetChatMembers]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, tt.want, resp.Body)
		})
	}
}

func TestNegativeChatHandler_GetChatMembers(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					GetAllChatMembers(gomock.Any(), int64(100), int64(1)).
					Return(nil, domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+tt.args.chatID+"/members", nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.GetChatMembers(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_QuitChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID int64
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "success quit chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					QuitChat(gomock.Any(), int64(101), int64(1)).
					Return(nil)

				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
					}, nil).AnyTimes()

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(3)
			},
			args: args{userID: 101, chatID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/quit", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.QuitChat(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[any]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
		})
	}
}

func TestNegativeChatHandler_QuitChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					QuitChat(gomock.Any(), int64(100), int64(1)).
					Return(domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
		{
			name: "owner cannot quit",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					QuitChat(gomock.Any(), int64(100), int64(1)).
					Return(domain.ErrOwnerCantQuitGroup)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.OwnerCantQuitChat,
		},
		{
			name: "cannot quit dialog",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					QuitChat(gomock.Any(), int64(100), int64(1)).
					Return(domain.ErrCantQuitDialog)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.CantQuitDialog,
		},
		{
			name: "chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					QuitChat(gomock.Any(), int64(100), int64(1)).
					Return(domain.ErrChatNotFound)
			},
			args:     args{userID: int64(100), chatID: "1"},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.CantFindChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+tt.args.chatID+"/quit", nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.QuitChat(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		title  string
		userID int64
		chatID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "success update chat title",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatTitle(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "New Title",
						LastMessage: dto.MessageDTO{
							Text:      "Hello",
							CreatedAt: time.Now(),
						},
						Avatar: strPtrChat("avatar.jpg"),
					}, nil)

				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "New Title",
						LastMessage: dto.MessageDTO{
							Text:      "Hello",
							CreatedAt: time.Now(),
						},
						Avatar: strPtrChat("avatar.jpg"),
					}, nil).AnyTimes()

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(2)
			},
			args: args{
				userID: 100,
				chatID: 1,
				title:  "New Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			reqBody := dto.RequestUpdateTitle{Title: tt.args.title}
			bodyBytes, _ := json.Marshal(reqBody)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/title", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.UpdateChatTitle(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, "New Title", resp.Body.Title)
		})
	}
}

func TestNegativeChatHandler_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
		body   string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1", body: `{"title":"New Title"}`},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:       "invalid json",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{invalid`},
			wantCode:   http.StatusBadRequest,
			wantErr:    dtoApi.InvalidJson,
			skipErrCmp: false,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid", body: `{"title":"New Title"}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name:       "validation fails - empty title",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{"title":""}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatTitle(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil, domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"title":"New Title"}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
		{
			name: "dialog cannot have custom title",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatTitle(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil, domain.ErrDialogCannotHaveCustomTitle)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"title":"New Title"}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.CantChangeTitle,
		},
		{
			name: "chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					UpdateChatTitle(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil, domain.ErrChatNotFound)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"title":"New Title"}`},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.CantFindChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+tt.args.chatID+"/title", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.UpdateChatTitle(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_AddMembersToChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		memberIDs []int64
		userID    int64
		chatID    int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "success add members to chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil)

				chatInfo := &dto.ChatInformationDTO{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "Group Chat",
				}
				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), int64(101)).
					Return(chatInfo, nil)
				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), int64(102)).
					Return(chatInfo, nil)
				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101, 102}, nil)
				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), int64(100)).
					Return(chatInfo, nil)

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(3)
			},
			args: args{
				userID:    100,
				chatID:    1,
				memberIDs: []int64{101, 102},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			reqBody := dto.RequestAddMember{MembersId: tt.args.memberIDs}
			bodyBytes, _ := json.Marshal(reqBody)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/members", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.AddMembersToChat(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[any]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
		})
	}
}

func TestNegativeChatHandler_AddMembersToChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
		body   string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1", body: `{"members_id":[101]}`},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:       "invalid json",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{invalid`},
			wantCode:   http.StatusBadRequest,
			wantErr:    dtoApi.InvalidJson,
			skipErrCmp: false,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid", body: `{"members_id":[101]}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name:       "validation fails - empty members list",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{"members_id":[]}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"members_id":[101]}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
		{
			name: "cannot add member to dialog",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrCantAddMemberToDialog)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"members_id":[101]}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.CantAddMemberToDialog,
		},
		{
			name: "only owner can add members",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(101), int64(1), gomock.Any()).
					Return(domain.ErrOnlyOwnerCanAddPeople)
			},
			args:     args{userID: int64(101), chatID: "1", body: `{"members_id":[102]}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.OnlyOwnerCanAddMembers,
		},
		{
			name: "member already in chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrMemberAlreadyInChat)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"members_id":[101]}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.MemberAlreadyInChat,
		},
		{
			name: "chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					AddMembersToChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrChatNotFound)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"members_id":[101]}`},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.CantFindChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+tt.args.chatID+"/members", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.AddMembersToChat(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveChatHandler_DeleteMemberFromChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID         int64
		chatID         int64
		memberToDelete int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "success delete member from chat",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(nil)

				f.chatService.EXPECT().
					GetChatMemberIDs(gomock.Any(), int64(1)).
					Return([]int64{100, 101}, nil)

				f.chatService.EXPECT().
					GetChatByID(gomock.Any(), int64(1), gomock.Any()).
					Return(&dto.ChatInformationDTO{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
					}, nil).AnyTimes()

				f.wsPublisher.EXPECT().
					PublishToUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return().
					Times(3)
			},
			args: args{
				userID:         100,
				chatID:         1,
				memberToDelete: 101,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			reqBody := dto.RequestDeleteMember{MemberId: tt.args.memberToDelete}
			bodyBytes, _ := json.Marshal(reqBody)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+strconv.FormatInt(tt.args.chatID, 10)+"/members", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strconv.FormatInt(tt.args.chatID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteMemberFromChat(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[any]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
		})
	}
}

func TestNegativeChatHandler_DeleteMemberFromChat(t *testing.T) {
	type fields struct {
		chatService *mock.MockChatService
		wsPublisher *mock.MockWsUserPublisher
	}

	type args struct {
		userID interface{}
		chatID string
		body   string
	}

	tests := []struct {
		prepare    func(*fields)
		args       args
		name       string
		wantErr    dtoApi.ErrorCode
		wantCode   int
		skipErrCmp bool
	}{
		{
			name:     "unauthorized - no userID",
			prepare:  nil,
			args:     args{userID: nil, chatID: "1", body: `{"member_id":101}`},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:       "invalid json",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{invalid`},
			wantCode:   http.StatusBadRequest,
			wantErr:    dtoApi.InvalidJson,
			skipErrCmp: false,
		},
		{
			name:     "invalid chat id",
			prepare:  nil,
			args:     args{userID: int64(100), chatID: "invalid", body: `{"member_id":101}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name:       "validation fails - missing member_id",
			prepare:    nil,
			args:       args{userID: int64(100), chatID: "1", body: `{}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrNotMember)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"member_id":101}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.NotMemberOfChat,
		},
		{
			name: "cannot delete member from dialog",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrCantDeleteMemberFromDialog)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"member_id":101}`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.DeleteMemberFromDialog,
		},
		{
			name: "only owner can delete members",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(101), int64(1), gomock.Any()).
					Return(domain.ErrOnlyOwnerCanDeletePeople)
			},
			args:     args{userID: int64(101), chatID: "1", body: `{"member_id":102}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.OnlyOwnerCanDeleteMember,
		},
		{
			name: "cannot delete owner",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrCantDeleteOwnerOfChat)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"member_id":100}`},
			wantCode: http.StatusForbidden,
			wantErr:  dtoApi.DeleteOwnerOfChat,
		},
		{
			name: "member not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrMemberNotFound)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"member_id":999}`},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.MemberNotFound,
		},
		{
			name: "chat not found",
			prepare: func(f *fields) {
				f.chatService.EXPECT().
					DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), gomock.Any()).
					Return(domain.ErrChatNotFound)
			},
			args:     args{userID: int64(100), chatID: "1", body: `{"member_id":101}`},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.CantFindChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatService: mock.NewMockChatService(ctrl),
				wsPublisher: mock.NewMockWsUserPublisher(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ChatsHandler{
				chatService: f.chatService,
				wsPublisher: f.wsPublisher,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+tt.args.chatID+"/members", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.args.chatID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteMemberFromChat(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorChat(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestNewChatHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "wires chat service and ws publisher"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc := mock.NewMockChatService(ctrl)
			ws := mock.NewMockWsUserPublisher(ctrl)
			h := NewChatHandler(svc, ws)
			require.NotNil(t, h)
			require.Equal(t, svc, h.chatService)
			require.Equal(t, ws, h.wsPublisher)
		})
	}
}

func strPtrChat(s string) *string {
	return &s
}
