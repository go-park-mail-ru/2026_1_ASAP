//go:generate mockgen -destination=mock/profile_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1 AuthClient
//go:generate mockgen -destination=mock/profile_client_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1 ProfileClient
package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/profile/mock"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveGatewayProfileHandler_GetMyProfile(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		userID  int64
	}{
		{
			name:   "Successful get my profile",
			userID: 100,
			prepare: func(f *fields) {
				f.profileClient.EXPECT().GetProfile(gomock.Any(), &profilev1.RequestGetProfile{
					UserId: 100,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
					LastName:  strPtr("Doe"),
					Bio:       "My bio",
					BirthDate: "1990-01-01",
					Avatar:    "avatar.jpg",
					LastSeen:  timestamppb.Now(),
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
					Email: "john@example.com",
				}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/me", handler.GetMyProfile)

			req := httptest.NewRequest(http.MethodGet, "/profile/me", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_GetMyProfile(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			userID: nil,
			want:   http.StatusUnauthorized,
		},
		{
			name:   "Invalid user_id type",
			userID: "invalid",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "Profile not found",
			userID: int64(999),
			prepare: func(f *fields) {
				f.profileClient.EXPECT().GetProfile(gomock.Any(), &profilev1.RequestGetProfile{
					UserId: 999,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "not found"))
			},
			want: http.StatusNotFound,
		},
		{
			name:   "Auth service error",
			userID: int64(100),
			prepare: func(f *fields) {
				f.profileClient.EXPECT().GetProfile(gomock.Any(), &profilev1.RequestGetProfile{
					UserId: 100,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(authv1.AuthErrorCode_AUTH_ERROR_USER_NOT_FOUND), "user not found"))
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/me", handler.GetMyProfile)

			req := httptest.NewRequest(http.MethodGet, "/profile/me", nil)
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

func TestPositiveGatewayProfileHandler_GetUserProfile(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		prepare   func(*fields)
		want      int
		name      string
		userID    int64
		profileID string
	}{
		{
			name:      "Successful get user profile",
			userID:    100,
			profileID: "200",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().GetProfile(gomock.Any(), &profilev1.RequestGetProfile{
					UserId: 200,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    200,
					FirstName: "Jane",
					LastName:  strPtr("Smith"),
					Bio:       "User bio",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 200,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "jane_smith",
					Email: "jane@example.com",
				}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/{id}", handler.GetUserProfile)

			req := httptest.NewRequest(http.MethodGet, "/profile/"+tt.profileID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_GetUserProfile(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name      string
		prepare   func(*fields)
		want      int
		userID    interface{}
		profileID string
	}{
		{
			name:      "Missing user_id",
			userID:    nil,
			profileID: "200",
			want:      http.StatusUnauthorized,
		},
		{
			name:      "Invalid profile id",
			userID:    int64(100),
			profileID: "invalid",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Profile not found",
			userID:    int64(100),
			profileID: "999",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().GetProfile(gomock.Any(), &profilev1.RequestGetProfile{
					UserId: 999,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "not found"))
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/{id}", handler.GetUserProfile)

			req := httptest.NewRequest(http.MethodGet, "/profile/"+tt.profileID, nil)
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

func TestPositiveGatewayProfileHandler_UpdateUserBio(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body dto.RequestUpdateBio
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
		userID  int64
	}{
		{
			name:   "Successful update bio",
			userID: 100,
			prepare: func(f *fields) {
				bio := "New bio"
				f.profileClient.EXPECT().UpdateProfileBio(gomock.Any(), &profilev1.RequestUpdateBio{
					UserId: 100,
					Bio:    &bio,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
					Bio:       "New bio",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
				}, nil)
			},
			args: args{
				body: dto.RequestUpdateBio{
					Bio: strPtr("New bio"),
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
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/bio", handler.UpdateUserBio)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/profile/bio", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_UpdateUserBio(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body interface{}
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			userID: nil,
			args: args{
				body: dto.RequestUpdateBio{Bio: strPtr("New bio")},
			},
			want: http.StatusUnauthorized,
		},
		{
			name:   "Invalid JSON body",
			userID: int64(100),
			args: args{
				body: "invalid json",
			},
			want: http.StatusBadRequest,
		},
		{
			name:   "Nil bio",
			userID: int64(100),
			args: args{
				body: dto.RequestUpdateBio{Bio: nil},
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/bio", handler.UpdateUserBio)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/profile/bio", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
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

func TestPositiveGatewayProfileHandler_UpdateUserBirthDate(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body dto.RequestUpdateBirthDate
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
		userID  int64
	}{
		{
			name:   "Successful update birth date",
			userID: 100,
			prepare: func(f *fields) {
				birthDate := "1990-01-01"
				f.profileClient.EXPECT().UpdateProfileBirthDate(gomock.Any(), &profilev1.RequestUpdateBirthDate{
					UserId:    100,
					BirthDate: &birthDate,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
					BirthDate: "1990-01-01",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
				}, nil)
			},
			args: args{
				body: dto.RequestUpdateBirthDate{
					BirthDate: strPtr("1990-01-01"),
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
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/birthdate", handler.UpdateUserBirthDate)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/profile/birthdate", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayProfileHandler_UpdateProfileName(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body dto.RequestUpdateName
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
		userID  int64
	}{
		{
			name:   "Successful update name without last name",
			userID: 100,
			prepare: func(f *fields) {
				f.profileClient.EXPECT().UpdateProfileName(gomock.Any(), &profilev1.RequestUpdateName{
					UserId:     100,
					FirstName:  "NewFirstName",
					SecondName: nil,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "NewFirstName",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
				}, nil)
			},
			args: args{
				body: dto.RequestUpdateName{
					FirstName: "NewFirstName",
					LastName:  nil,
				},
			},
			want: http.StatusOK,
		},
		{
			name:   "Successful update name with last name",
			userID: 100,
			prepare: func(f *fields) {
				lastName := "NewLastName"
				f.profileClient.EXPECT().UpdateProfileName(gomock.Any(), &profilev1.RequestUpdateName{
					UserId:     100,
					FirstName:  "NewFirstName",
					SecondName: &lastName,
				}).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "NewFirstName",
					LastName:  &lastName,
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
				}, nil)
			},
			args: args{
				body: dto.RequestUpdateName{
					FirstName: "NewFirstName",
					LastName:  strPtr("NewLastName"),
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
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/name", handler.UpdateProfileName)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/profile/name", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_UpdateProfileName(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body interface{}
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			userID: nil,
			args: args{
				body: dto.RequestUpdateName{FirstName: "John"},
			},
			want: http.StatusUnauthorized,
		},
		{
			name:   "Invalid JSON body",
			userID: int64(100),
			args: args{
				body: "invalid json",
			},
			want: http.StatusBadRequest,
		},
		{
			name:   "Empty first name",
			userID: int64(100),
			args: args{
				body: dto.RequestUpdateName{FirstName: ""},
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/name", handler.UpdateProfileName)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/profile/name", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
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

func TestPositiveGatewayProfileHandler_SearchIdByLogin(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		login   string
	}{
		{
			name:  "Successful search by login",
			login: "john_doe",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().SearchIdByLogin(gomock.Any(), &profilev1.RequestSearchIdByLogin{
					Login: "john_doe",
				}).Return(&profilev1.ResponseSearchIdByLogin{
					UserId: 100,
					Login:  "john_doe",
				}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/search", handler.SearchIdByLogin)

			req := httptest.NewRequest(http.MethodGet, "/profile/search?login="+tt.login, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_SearchIdByLogin(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		login   string
	}{
		{
			name:  "Empty login",
			login: "",
			want:  http.StatusBadRequest,
		},
		{
			name:  "Login with spaces only",
			login: "",
			want:  http.StatusBadRequest,
		},
		{
			name:  "User not found",
			login: "nonexistent",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().SearchIdByLogin(gomock.Any(), &profilev1.RequestSearchIdByLogin{
					Login: "nonexistent",
				}).Return(nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "not found"))
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/profile/search", handler.SearchIdByLogin)

			req := httptest.NewRequest(http.MethodGet, "/profile/search?login="+tt.login, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayProfileHandler_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		userID  int64
	}{
		{
			name:   "Successful update avatar",
			userID: 100,
			prepare: func(f *fields) {
				f.profileClient.EXPECT().UpdateProfileAvatar(gomock.Any(), gomock.Any()).Return(&profilev1.ResponseGetProfile{
					UserId:    100,
					FirstName: "John",
					Avatar:    "new_avatar.jpg",
				}, nil)

				f.authClient.EXPECT().GetUserPublic(gomock.Any(), &authv1.RequestGetUserPublic{
					UserId: 100,
				}).Return(&authv1.ResponseGetUserPublic{
					Login: "john_doe",
				}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/avatar", handler.UpdateUserAvatar)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("avatar", "avatar.jpg")
			part.Write([]byte("fake image data"))
			writer.Close()

			req := httptest.NewRequest(http.MethodPut, "/profile/avatar", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayProfileHandler_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		authClient    *mock.MockAuthClient
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			userID: nil,
			want:   http.StatusUnauthorized,
		},
		{
			name:   "No file uploaded",
			userID: int64(100),
			want:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authClient:    mock.NewMockAuthClient(ctrl),
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayProfileHandler{
				AuthService:    f.authClient,
				ProfileService: f.profileClient,
			}

			r := chi.NewRouter()
			r.Put("/profile/avatar", handler.UpdateUserAvatar)

			req := httptest.NewRequest(http.MethodPut, "/profile/avatar", nil)
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
