package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const (
	validLoginJSON    = `{"login":"alice","password":"Valid1!pass"}`
	validRegisterJSON = `{"login":"alice","email":"alice@example.com","password":"Valid1!pass"}`
)

func decodeAPIError(t *testing.T, rr *httptest.ResponseRecorder) dtoApi.ApiErrorResponse {
	t.Helper()
	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	return got
}

func TestPositiveAuthHandler_Login(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		body string
	}

	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "200 sets cookie and CSRF header",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Login(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestLogin{})).
					Return(&dtoSession.SessionDTO{
						SessionID: "sid-1",
						CSRFToken: "csrf-1",
						Expire:    exp,
					}, nil)
			},
			args: args{body: validLoginJSON},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			h.Login(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, "csrf-1", rr.Header().Get("X-NEW-CSRF-TOKEN"))

			var cookies []*http.Cookie
			for _, c := range rr.Result().Cookies() {
				if c.Name == "session_id" {
					cookies = append(cookies, c)
				}
			}
			require.Len(t, cookies, 1)
			require.Equal(t, "sid-1", cookies[0].Value)
			require.True(t, cookies[0].HttpOnly)
			require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)

			var ok dtoApi.ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, dtoApi.Success, ok.Status)
			require.Equal(t, "alice", ok.Body.Login)
		})
	}
}

func TestNegativeAuthHandler_Login(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		body string
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantCode   int
		wantCodeIn dtoApi.ErrorCode
	}{
		{
			name:       "invalid JSON",
			prepare:    nil,
			args:       args{body: `{`},
			wantCode:   http.StatusBadRequest,
			wantCodeIn: dtoApi.InvalidJson,
		},
		{
			name:       "validation fails",
			prepare:    nil,
			args:       args{body: `{"login":"ab","password":"Valid1!pass"}`},
			wantCode:   http.StatusBadRequest,
			wantCodeIn: "", // mapped from validation; any error in list
		},
		{
			name: "invalid credentials",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Login(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestLogin{})).
					Return(nil, domainUser.ErrInvalidCredentials)
			},
			args:       args{body: validLoginJSON},
			wantCode:   http.StatusUnauthorized,
			wantCodeIn: dtoApi.InvalidCredentials,
		},
		{
			name: "user not found",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Login(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestLogin{})).
					Return(nil, domainUser.ErrNotFound)
			},
			args:       args{body: validLoginJSON},
			wantCode:   http.StatusUnauthorized,
			wantCodeIn: dtoApi.InvalidCredentials,
		},
		{
			name: "internal service error",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Login(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestLogin{})).
					Return(nil, errors.New("db down"))
			},
			args:       args{body: validLoginJSON},
			wantCode:   http.StatusInternalServerError,
			wantCodeIn: dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			h.Login(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.NotEmpty(t, got.Errors)
			if tt.wantCodeIn != "" {
				require.Equal(t, tt.wantCodeIn, got.Errors[0].Code)
			}
		})
	}
}

func TestPositiveAuthHandler_Register(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		body string
	}

	exp := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "200 sets cookie and CSRF header",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Register(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestRegistrate{})).
					Return(&dtoSession.SessionDTO{
						SessionID: "sid-reg",
						CSRFToken: "csrf-reg",
						Expire:    exp,
					}, nil)
			},
			args: args{body: validRegisterJSON},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			h.Register(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, "csrf-reg", rr.Header().Get("X-NEW-CSRF-TOKEN"))

			var ok dtoApi.ApiSuccessResponse[dtoAuth.ResponseRegisterSuccess]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, dtoApi.Success, ok.Status)
			require.Equal(t, "alice", ok.Body.Login)
			require.Equal(t, "alice@example.com", ok.Body.Email)
		})
	}
}

func TestNegativeAuthHandler_Register(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		body string
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantCode   int
		wantErr    dtoApi.ErrorCode
		skipErrCmp bool
	}{
		{
			name:     "invalid JSON",
			prepare:  nil,
			args:     args{body: `{`},
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidJson,
		},
		{
			name:       "validation fails",
			prepare:    nil,
			args:       args{body: `{"login":"ab","email":"x@y.com","password":"Valid1!pass"}`},
			wantCode:   http.StatusBadRequest,
			skipErrCmp: true,
		},
		{
			name: "login already exists",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Register(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestRegistrate{})).
					Return(nil, domainUser.ErrLoginAlreadyExists)
			},
			args:     args{body: validRegisterJSON},
			wantCode: http.StatusConflict,
			wantErr:  dtoApi.LoginAlreadyRegistered,
		},
		{
			name: "email already exists",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Register(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestRegistrate{})).
					Return(nil, domainUser.ErrEmailAlreadyExists)
			},
			args:     args{body: validRegisterJSON},
			wantCode: http.StatusConflict,
			wantErr:  dtoApi.EmailAlreadyRegistered,
		},
		{
			name: "internal service error",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Register(gomock.Any(), gomock.AssignableToTypeOf(&dtoAuth.RequestRegistrate{})).
					Return(nil, errors.New("db"))
			},
			args:     args{body: validRegisterJSON},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.args.body))
			req.Header.Set("Content-Type", "application/json")

			h.Register(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.NotEmpty(t, got.Errors)
			if !tt.skipErrCmp {
				require.Equal(t, tt.wantErr, got.Errors[0].Code)
			}
		})
	}
}

func TestPositiveAuthHandler_Logout(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		sessionID string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "200 on successful logout",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Logout(gomock.Any(), &dtoAuth.RequestLogout{SessionID: "cookie-sid"}).
					Return(nil)
			},
			args: args{sessionID: "cookie-sid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			ctx := context.WithValue(req.Context(), middleware.SessionID, tt.args.sessionID)
			req = req.WithContext(ctx)

			h.Logout(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var ok dtoApi.ApiSuccessResponse[dtoAuth.ResponseLogoutSuccess]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, dtoApi.Success, ok.Status)
		})
	}
}

func TestNegativeAuthHandler_Logout(t *testing.T) {
	type fields struct {
		authService *mock.MockAuthService
	}

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name     string
		prepare  func(*fields)
		args     args
		wantCode int
		wantErr  dtoApi.ErrorCode
	}{
		{
			name:     "missing session in context",
			prepare:  nil,
			args:     args{ctx: context.Background()},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:    "wrong context value type",
			prepare: nil,
			args: args{
				ctx: context.WithValue(context.Background(), middleware.SessionID, 123),
			},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "session not found",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Logout(gomock.Any(), &dtoAuth.RequestLogout{SessionID: "gone"}).
					Return(domainSession.ErrNotFound)
			},
			args: args{
				ctx: context.WithValue(context.Background(), middleware.SessionID, "gone"),
			},
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.authService.EXPECT().
					Logout(gomock.Any(), &dtoAuth.RequestLogout{SessionID: "sid"}).
					Return(errors.New("redis"))
			},
			args: args{
				ctx: context.WithValue(context.Background(), middleware.SessionID, "sid"),
			},
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{authService: mock.NewMockAuthService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			h := &AuthHandler{AuthService: f.authService}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			req = req.WithContext(tt.args.ctx)

			h.Logout(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.NotEmpty(t, got.Errors)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestNewAuthHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := mock.NewMockAuthService(ctrl)
	h := NewAuthHandler(svc)
	require.NotNil(t, h)
	require.Equal(t, svc, h.AuthService)
}
