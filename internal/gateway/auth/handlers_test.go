//go:generate mockgen -destination=mock/handlers_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1 AuthClient
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/auth/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func TestPositiveGatewayAuthHandler_Login(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
	}

	type args struct {
		body map[string]string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful login with login",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Login(gomock.Any(), &authv1.RequestLogin{
					Login:    "testuser123",
					Password: "Password123!",
				}).Return(&authv1.ResponseLogin{
					Session: &authv1.SessionMeta{
						SessionId: "session123",
						CsrfToken: "csrf123",
						UserId:    100,
					},
					Login: "testuser123",
				}, nil)
			},
			args: args{
				body: map[string]string{
					"login":    "testuser123",
					"password": "Password123!",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful login with email",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Login(gomock.Any(), &authv1.RequestLogin{
					Login:    "test@example.com",
					Password: "Password123!",
				}).Return(&authv1.ResponseLogin{
					Session: &authv1.SessionMeta{
						SessionId: "session456",
						CsrfToken: "csrf456",
						UserId:    200,
					},
					Login: "testuser",
				}, nil)
			},
			args: args{
				body: map[string]string{
					"login":    "test@example.com",
					"password": "Password123!",
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
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/login", handler.Login)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayAuthHandler_Login(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
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
			name: "Missing login field",
			args: args{
				body: map[string]string{
					"password": "Password123!",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing password field",
			args: args{
				body: map[string]string{
					"login": "testuser123",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Empty login",
			args: args{
				body: map[string]string{
					"login":    "",
					"password": "Password123!",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Empty password",
			args: args{
				body: map[string]string{
					"login":    "testuser123",
					"password": "",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid credentials - wrong password",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Login(gomock.Any(), &authv1.RequestLogin{
					Login:    "testuser123",
					Password: "WrongPass123!",
				}).Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_INVALID_CREDENTIALS), "invalid credentials"))
			},
			args: args{
				body: map[string]string{
					"login":    "testuser123",
					"password": "WrongPass123!",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Login(gomock.Any(), &authv1.RequestLogin{
					Login:    "nonexistent",
					Password: "Password123!",
				}).Return(nil, grpcerr.New(codes.NotFound, int32(authv1.AuthErrorCode_AUTH_ERROR_USER_NOT_FOUND), "user not found"))
			},
			args: args{
				body: map[string]string{
					"login":    "nonexistent",
					"password": "Password123!",
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
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/login", handler.Login)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayAuthHandler_Register(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
	}

	type args struct {
		body map[string]string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful registration",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Register(gomock.Any(), &authv1.RequestRegister{
					Login:    "newuser123",
					Email:    "newuser@example.com",
					Password: "Password123!",
				}).Return(&authv1.ResponseRegister{
					Session: &authv1.SessionMeta{
						SessionId: "session789",
						CsrfToken: "csrf789",
						UserId:    300,
					},
					Login: "newuser123",
					Email: "newuser@example.com",
				}, nil)
			},
			args: args{
				body: map[string]string{
					"login":    "newuser123",
					"email":    "newuser@example.com",
					"password": "Password123!",
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
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/register", handler.Register)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayAuthHandler_Register(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
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
			name: "Missing login field",
			args: args{
				body: map[string]string{
					"email":    "test@example.com",
					"password": "Password123!",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing email field",
			args: args{
				body: map[string]string{
					"login":    "testuser123",
					"password": "Password123!",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing password field",
			args: args{
				body: map[string]string{
					"login": "testuser123",
					"email": "test@example.com",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Invalid email format",
			args: args{
				body: map[string]string{
					"login":    "testuser123",
					"email":    "invalid-email",
					"password": "Password123!",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Login already exists",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Register(gomock.Any(), &authv1.RequestRegister{
					Login:    "existinguser",
					Email:    "new@example.com",
					Password: "Password123!",
				}).Return(nil, grpcerr.New(codes.AlreadyExists, int32(authv1.AuthErrorCode_AUTH_ERROR_LOGIN_ALREADY_EXISTS), "login already exists"))
			},
			args: args{
				body: map[string]string{
					"login":    "existinguser",
					"email":    "new@example.com",
					"password": "Password123!",
				},
			},
			want: http.StatusConflict,
		},
		{
			name: "Email already exists",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Register(gomock.Any(), &authv1.RequestRegister{
					Login:    "newuser123",
					Email:    "existing@example.com",
					Password: "Password123!",
				}).Return(nil, grpcerr.New(codes.AlreadyExists, int32(authv1.AuthErrorCode_AUTH_ERROR_EMAIL_ALREADY_EXISTS), "email already exists"))
			},
			args: args{
				body: map[string]string{
					"login":    "newuser123",
					"email":    "existing@example.com",
					"password": "Password123!",
				},
			},
			want: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/register", handler.Register)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayAuthHandler_Logout(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
	}

	type args struct {
		cookieValue string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful logout",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Logout(gomock.Any(), &authv1.RequestLogout{
					SessionId: "session123",
				}).Return(&authv1.ResponseLogout{}, nil)
			},
			args: args{
				cookieValue: "session123",
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/logout", handler.Logout)

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			if tt.args.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.args.cookieValue,
				})
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayAuthHandler_Logout(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
	}

	type args struct {
		cookieValue string
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name:        "No session cookie",
			args:        args{cookieValue: ""},
			want:        http.StatusUnauthorized,
		},
		{
			name:        "Empty session cookie",
			args:        args{cookieValue: ""},
			want:        http.StatusUnauthorized,
		},
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Logout(gomock.Any(), &authv1.RequestLogout{
					SessionId: "invalid_session",
				}).Return(nil, grpcerr.New(codes.NotFound, int32(authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND), "session not found"))
			},
			args: args{cookieValue: "invalid_session"},
			want: http.StatusUnauthorized,
		},
		{
			name: "Internal server error",
			prepare: func(f *fields) {
				f.authClient.EXPECT().Logout(gomock.Any(), &authv1.RequestLogout{
					SessionId: "session123",
				}).Return(nil, grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "internal error"))
			},
			args: args{cookieValue: "session123"},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/logout", handler.Logout)

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			if tt.args.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: tt.args.cookieValue,
				})
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayAuthHandler_VkIDLogin(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
	}

	type args struct {
		body map[string]string
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful VK ID login",
			prepare: func(f *fields) {
				f.authClient.EXPECT().AuthVKID(gomock.Any(), &authv1.RequestVKID{
					Code:         "auth_code_123",
					State:        "state_456",
					CodeVerifier: "verifier_789",
					DeviceId:     "device_001",
				}).Return(&authv1.ResponseLogin{
					Session: &authv1.SessionMeta{
						SessionId: "vk_session_123",
						CsrfToken: "vk_csrf_456",
						UserId:    500,
					},
					Login: "vk_user",
				}, nil)
			},
			args: args{
				body: map[string]string{
					"code":          "auth_code_123",
					"state":         "state_456",
					"code_verifier": "verifier_789",
					"device_id":     "device_001",
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
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/auth/vk", handler.VkIDLogin)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/vk", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayAuthHandler_VkIDLogin(t *testing.T) {
	type fields struct {
		authClient *mock.MockAuthClient
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
			name: "Invalid credentials from VK",
			prepare: func(f *fields) {
				f.authClient.EXPECT().AuthVKID(gomock.Any(), &authv1.RequestVKID{
					Code:         "invalid_code",
					State:        "state_456",
					CodeVerifier: "verifier_789",
					DeviceId:     "device_001",
				}).Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_INVALID_CREDENTIALS), "invalid credentials"))
			},
			args: args{
				body: map[string]string{
					"code":          "invalid_code",
					"state":         "state_456",
					"code_verifier": "verifier_789",
					"device_id":     "device_001",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "VK service error",
			prepare: func(f *fields) {
				f.authClient.EXPECT().AuthVKID(gomock.Any(), &authv1.RequestVKID{
					Code:         "auth_code_123",
					State:        "state_456",
					CodeVerifier: "verifier_789",
					DeviceId:     "device_001",
				}).Return(nil, grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "vk service error"))
			},
			args: args{
				body: map[string]string{
					"code":          "auth_code_123",
					"state":         "state_456",
					"code_verifier": "verifier_789",
					"device_id":     "device_001",
				},
			},
			want: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient: mock.NewMockAuthClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAuthHandler{
				AuthService: f.authClient,
			}

			r := chi.NewRouter()
			r.Post("/auth/vk", handler.VkIDLogin)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/vk", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}