package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/contacts/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
)

func strPtrContact(s string) *string {
	return &s
}

func decodeAPIErrorContact(t *testing.T, rr *httptest.ResponseRecorder) dtoApi.ApiErrorResponse {
	t.Helper()
	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	return got
}

func TestPositiveContactHandler_GetContacts(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
	}

	type args struct {
		userID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		want    []*dto.ContactResponse
		args    args
	}{
		{
			name: "success get contacts",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					GetContacts(gomock.Any(), int64(100)).
					Return([]*dto.ContactResponse{
						{
							UserID:           100,
							ContactUserID:    200,
							FirstName:        "Alice",
							LastName:         strPtrContact("Smith"),
							ContactAvatarUrl: strPtrContact("avatar.jpg"),
						},
						{
							UserID:           100,
							ContactUserID:    300,
							FirstName:        "Bob",
							LastName:         nil,
							ContactAvatarUrl: nil,
						},
					}, nil)
			},
			args: args{userID: 100},
			want: []*dto.ContactResponse{
				{
					UserID:           100,
					ContactUserID:    200,
					FirstName:        "Alice",
					LastName:         strPtrContact("Smith"),
					ContactAvatarUrl: strPtrContact("avatar.jpg"),
				},
				{
					UserID:           100,
					ContactUserID:    300,
					FirstName:        "Bob",
					LastName:         nil,
					ContactAvatarUrl: nil,
				},
			},
		},
		{
			name: "success get empty contacts",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					GetContacts(gomock.Any(), int64(100)).
					Return([]*dto.ContactResponse{}, nil)
			},
			args: args{userID: 100},
			want: []*dto.ContactResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			h.GetContacts(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[[]*dto.ContactResponse]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, len(tt.want), len(resp.Body))

			for i, contact := range tt.want {
				require.Equal(t, contact.UserID, resp.Body[i].UserID)
				require.Equal(t, contact.ContactUserID, resp.Body[i].ContactUserID)
				require.Equal(t, contact.FirstName, resp.Body[i].FirstName)
				require.Equal(t, contact.LastName, resp.Body[i].LastName)
				require.Equal(t, contact.ContactAvatarUrl, resp.Body[i].ContactAvatarUrl)
			}
		})
	}
}

func TestNegativeContactHandler_GetContacts(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
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
				f.contactService.EXPECT().
					GetContacts(gomock.Any(), int64(100)).
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
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			h.GetContacts(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorContact(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveContactHandler_CreateContact(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
	}

	type args struct {
		lastName      *string
		firstName     string
		userID        int64
		contactUserID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ContactResponse
		name    string
		args    args
	}{
		{
			name: "success create contact with last name",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(&dto.ContactResponse{
						UserID:           100,
						ContactUserID:    200,
						FirstName:        "Alice",
						LastName:         strPtrContact("Smith"),
						ContactAvatarUrl: strPtrContact("avatar.jpg"),
					}, nil)
			},
			args: args{
				userID:        100,
				contactUserID: 200,
				firstName:     "Alice",
				lastName:      strPtrContact("Smith"),
			},
			want: &dto.ContactResponse{
				UserID:           100,
				ContactUserID:    200,
				FirstName:        "Alice",
				LastName:         strPtrContact("Smith"),
				ContactAvatarUrl: strPtrContact("avatar.jpg"),
			},
		},
		{
			name: "success create contact without last name",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(&dto.ContactResponse{
						UserID:           100,
						ContactUserID:    300,
						FirstName:        "Bob",
						LastName:         nil,
						ContactAvatarUrl: nil,
					}, nil)
			},
			args: args{
				userID:        100,
				contactUserID: 300,
				firstName:     "Bob",
				lastName:      nil,
			},
			want: &dto.ContactResponse{
				UserID:           100,
				ContactUserID:    300,
				FirstName:        "Bob",
				LastName:         nil,
				ContactAvatarUrl: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			reqBody := dto.AddContactRequest{
				ContactUserID: tt.args.contactUserID,
				FirstName:     tt.args.firstName,
				LastName:      tt.args.lastName,
			}
			bodyBytes, err := json.Marshal(reqBody)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			h.CreateContact(rr, req)

			require.Equal(t, http.StatusCreated, rr.Code)

			var resp dtoApi.ApiSuccessResponse[*dto.ContactResponse]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, tt.want.UserID, resp.Body.UserID)
			require.Equal(t, tt.want.ContactUserID, resp.Body.ContactUserID)
			require.Equal(t, tt.want.FirstName, resp.Body.FirstName)
			require.Equal(t, tt.want.LastName, resp.Body.LastName)
			require.Equal(t, tt.want.ContactAvatarUrl, resp.Body.ContactAvatarUrl)
		})
	}
}

func TestNegativeContactHandler_CreateContact(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
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
			name:       "validation fails - missing contact_user_id",
			prepare:    nil,
			args:       args{userID: int64(100), body: `{"first_name":"Alice"}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "user not found",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, domainUser.ErrNotFound)
			},
			args: args{
				userID: int64(100),
				body:   `{"contact_user_id":200,"first_name":"Alice"}`,
			},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.InvalidCredentials,
		},
		{
			name: "contact already exists",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, domain.ErrContactExists)
			},
			args: args{
				userID: int64(100),
				body:   `{"contact_user_id":200,"first_name":"Alice"}`,
			},
			wantCode: http.StatusConflict,
			wantErr:  dtoApi.ContactAlreadyExists,
		},
		{
			name: "cannot create contact with yourself",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, domain.ErrCantCreateContactWithYourself)
			},
			args: args{
				userID: int64(100),
				body:   `{"contact_user_id":100,"first_name":"Alice"}`,
			},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.ContactWithYourself,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					AddContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil, errors.New("db error"))
			},
			args: args{
				userID: int64(100),
				body:   `{"contact_user_id":200,"first_name":"Alice"}`,
			},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			h.CreateContact(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			// Для успешных кейсов не проверяем ошибку
			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorContact(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestPositiveContactHandler_DeleteContact(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
	}

	type args struct {
		userID        int64
		contactUserID int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "success delete contact",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					DeleteContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(nil)
			},
			args: args{userID: 100, contactUserID: 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/"+strconv.FormatInt(tt.args.contactUserID, 10), nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("contact_user_id", strconv.FormatInt(tt.args.contactUserID, 10))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteContact(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var resp dtoApi.ApiSuccessResponse[any]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, dtoApi.Success, resp.Status)
			require.Equal(t, "Contact successful delete", resp.Body)
		})
	}
}

func TestNegativeContactHandler_DeleteContact(t *testing.T) {
	type fields struct {
		contactService *mock.MockContactService
	}

	type args struct {
		userID        interface{}
		contactUserID string
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
			args:     args{userID: nil, contactUserID: "200"},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid contact user id",
			prepare:  nil,
			args:     args{userID: int64(100), contactUserID: "invalid"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name:     "contact user id less than 1",
			prepare:  nil,
			args:     args{userID: int64(100), contactUserID: "0"},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "user not found",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					DeleteContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(domainUser.ErrNotFound)
			},
			args:     args{userID: int64(100), contactUserID: "200"},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.InvalidCredentials,
		},
		{
			name: "contact not found",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					DeleteContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(domain.ErrContactNotFound)
			},
			args:     args{userID: int64(100), contactUserID: "200"},
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.ContactNotFound,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.contactService.EXPECT().
					DeleteContact(gomock.Any(), gomock.Any(), int64(100)).
					Return(errors.New("db error"))
			},
			args:     args{userID: int64(100), contactUserID: "200"},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				contactService: mock.NewMockContactService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &ContactHandler{
				contactService: f.contactService,
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/"+tt.args.contactUserID, nil)

			if uid, ok := tt.args.userID.(int64); ok {
				ctx := context.WithValue(req.Context(), middleware.UserID, uid)
				req = req.WithContext(ctx)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("contact_user_id", tt.args.contactUserID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			h.DeleteContact(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			if tt.wantCode >= 400 && tt.wantCode < 500 {
				got := decodeAPIErrorContact(t, rr)
				require.NotEmpty(t, got.Errors)
				if !tt.skipErrCmp && tt.wantErr != "" {
					require.Equal(t, tt.wantErr, got.Errors[0].Code)
				}
			}
		})
	}
}

func TestNewContactHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := mock.NewMockContactService(ctrl)
	h := NewContactHandler(svc)
	require.NotNil(t, h)
	require.Equal(t, svc, h.contactService)
}
