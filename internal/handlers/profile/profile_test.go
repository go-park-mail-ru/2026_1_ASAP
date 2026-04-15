package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/profile/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
)

var testAvatarPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func strPtr(s string) *string { return &s }

func decodeAPIError(t *testing.T, rr *httptest.ResponseRecorder) dtoApi.ApiErrorResponse {
	t.Helper()
	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	return got
}

func withUserID(ctx context.Context, uid int64) context.Context {
	return context.WithValue(ctx, middleware.UserID, uid)
}

func withRouteParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func newMultipartAvatarRequest(t *testing.T, fileName string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("avatar", fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/me/avatar", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestPositiveProfileHandler_GetMyProfile(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	body := dto.ResponseGetProfile{
		UserId: 9, Login: "u9", FirstName: "U", Email: "u9@x.test",
	}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200 returns profile",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(9)).Return(&body, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/me", nil)
			req = req.WithContext(withUserID(req.Context(), 9))

			h.GetMyProfile(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var ok dtoApi.ApiSuccessResponse[dto.ResponseGetProfile]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, dtoApi.Success, ok.Status)
			require.Equal(t, body.UserId, ok.Body.UserId)
			require.Equal(t, body.Login, ok.Body.Login)
		})
	}
}

func TestNegativeProfileHandler_GetMyProfile(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "no user id",
			prepare:  nil,
			ctx:      context.Background(),
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "profile not found",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(1)).Return(nil, domain.ErrNotFound)
			},
			ctx:      withUserID(context.Background(), 1),
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "internal error",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(2)).Return(nil, errors.New("db"))
			},
			ctx:      withUserID(context.Background(), 2),
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			req = req.WithContext(tt.ctx)

			h.GetMyProfile(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_GetUserProfile(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	body := dto.ResponseGetProfile{UserId: 3, Login: "p3", FirstName: "P", Email: "p3@x.test"}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(3)).Return(&body, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/3", nil)
			req = req.WithContext(withUserID(req.Context(), 9))
			req = withRouteParam(req, "id", "3")

			h.GetUserProfile(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var ok dtoApi.ApiSuccessResponse[dto.ResponseGetProfile]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, int64(3), ok.Body.UserId)
		})
	}
}

func TestNegativeProfileHandler_GetUserProfile(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		idParam  string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "unauthorized",
			prepare:  nil,
			ctx:      context.Background(),
			idParam:  "1",
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid id",
			prepare:  nil,
			ctx:      withUserID(context.Background(), 1),
			idParam:  "nan",
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidID,
		},
		{
			name: "not found",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(99)).Return(nil, domain.ErrNotFound)
			},
			ctx:      withUserID(context.Background(), 1),
			idParam:  "99",
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.NotFound,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().GetUserProfile(gomock.Any(), int64(5)).Return(nil, errors.New("e"))
			},
			ctx:      withUserID(context.Background(), 1),
			idParam:  "5",
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/"+tt.idParam, nil)
			req = req.WithContext(tt.ctx)
			req = withRouteParam(req, "id", tt.idParam)

			h.GetUserProfile(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_UpdateUserBio(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	upd := dto.ResponseUpdateProfile{UserId: 1, Login: "a", FirstName: "A"}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBio(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBio{})).
					Return(&upd, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/bio", strings.NewReader(`{"bio":"hello"}`))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withUserID(req.Context(), 1))

			h.UpdateUserBio(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var ok dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, "A", ok.Body.FirstName)
		})
	}
}

func TestNegativeProfileHandler_UpdateUserBio(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		body     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "unauthorized",
			prepare:  nil,
			ctx:      context.Background(),
			body:     `{"bio":"x"}`,
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid json",
			prepare:  nil,
			ctx:      withUserID(context.Background(), 1),
			body:     `{`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidJson,
		},
		{
			name: "empty bio",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBio(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBio{})).
					Return(nil, domain.ErrEmptyBio)
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"bio":"text"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.EmptyBIO,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBio(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBio{})).
					Return(nil, errors.New("e"))
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"bio":"text"}`,
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/bio", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.ctx)

			h.UpdateUserBio(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_UpdateProfileBirthDate(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	upd := dto.ResponseUpdateProfile{UserId: 1, Login: "a", FirstName: "A", BirthDate: strPtr("1990-01-01")}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBirthDate(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBirthDate{})).
					Return(&upd, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/birth", strings.NewReader(`{"birth_date":"1990-01-01"}`))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withUserID(req.Context(), 1))

			h.UpdateProfileBirthDate(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestNegativeProfileHandler_UpdateProfileBirthDate(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		body     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "unauthorized",
			prepare:  nil,
			ctx:      context.Background(),
			body:     `{"birth_date":"1990-01-01"}`,
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid json",
			prepare:  nil,
			ctx:      withUserID(context.Background(), 1),
			body:     `{`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidJson,
		},
		{
			name: "invalid date format",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBirthDate(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBirthDate{})).
					Return(nil, domain.ErrInvalidBirthDateFormat)
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"birth_date":"bad"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidDateFormat,
		},
		{
			name: "invalid date",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBirthDate(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBirthDate{})).
					Return(nil, domain.ErrInvalidBirthDate)
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"birth_date":"2099-01-01"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidDate,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileBirthDate(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateBirthDate{})).
					Return(nil, errors.New("e"))
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"birth_date":"1990-01-01"}`,
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/birth", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.ctx)

			h.UpdateProfileBirthDate(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_SearchIdByLogin(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	res := dto.ResponseSearchIdByLogin{UserId: 42, Login: "bob"}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					SearchIdByLogin(gomock.Any(), &dto.RequestSearchIdByLogin{Login: "bob"}).
					Return(&res, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/search?login=bob", nil)

			h.SearchIdByLogin(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			var ok dtoApi.ApiSuccessResponse[dto.ResponseSearchIdByLogin]
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ok))
			require.Equal(t, int64(42), ok.Body.UserId)
		})
	}
}

func TestNegativeProfileHandler_SearchIdByLogin(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		prepare  func(*fields)
		name     string
		query    string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "missing login",
			prepare:  nil,
			query:    "",
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidJson,
		},
		{
			name: "not found",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					SearchIdByLogin(gomock.Any(), &dto.RequestSearchIdByLogin{Login: "nobody"}).
					Return(nil, domain.ErrNotFound)
			},
			query:    "nobody",
			wantCode: http.StatusNotFound,
			wantErr:  dtoApi.NotFound,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					SearchIdByLogin(gomock.Any(), &dto.RequestSearchIdByLogin{Login: "x"}).
					Return(nil, errors.New("e"))
			},
			query:    "x",
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			url := "/search"
			if tt.query != "" {
				url += "?login=" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			h.SearchIdByLogin(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_UpdateProfileName(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	upd := dto.ResponseUpdateProfile{UserId: 1, Login: "a", FirstName: "Ann"}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileName(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateName{})).
					Return(&upd, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/name", strings.NewReader(`{"first_name":"Ann"}`))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(withUserID(req.Context(), 1))

			h.UpdateProfileName(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestNegativeProfileHandler_UpdateProfileName(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		body     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "unauthorized",
			prepare:  nil,
			ctx:      context.Background(),
			body:     `{"first_name":"A"}`,
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:     "invalid json",
			prepare:  nil,
			ctx:      withUserID(context.Background(), 1),
			body:     `{`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidJson,
		},
		{
			name: "empty first name",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileName(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateName{})).
					Return(nil, domain.ErrEmptyFirstName)
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"first_name":"X"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.EmptyFirstName,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileName(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateName{})).
					Return(nil, errors.New("e"))
			},
			ctx:      withUserID(context.Background(), 1),
			body:     `{"first_name":"Ann"}`,
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/name", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.ctx)

			h.UpdateProfileName(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	upd := dto.ResponseUpdateProfile{UserId: 1, Login: "a", FirstName: "A", Avatar: strPtr("https://cdn/x.png")}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileAvatar(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateAvatar{})).
					Return(&upd, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := newMultipartAvatarRequest(t, "a.png", testAvatarPNG)
			req = req.WithContext(withUserID(req.Context(), 1))

			h.UpdateUserAvatar(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestNegativeProfileHandler_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		buildReq func(*testing.T) *http.Request
		name     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:    "unauthorized",
			prepare: nil,
			buildReq: func(t *testing.T) *http.Request {
				return newMultipartAvatarRequest(t, "a.png", testAvatarPNG)
			},
			ctx:      context.Background(),
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name:    "missing avatar field",
			prepare: nil,
			buildReq: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/", nil)
			},
			ctx:      withUserID(context.Background(), 1),
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.FileNotFound,
		},
		{
			name: "service invalid type",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileAvatar(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateAvatar{})).
					Return(nil, domain.ErrInvalidAvatarType)
			},
			buildReq: func(t *testing.T) *http.Request {
				return newMultipartAvatarRequest(t, "a.png", testAvatarPNG)
			},
			ctx:      withUserID(context.Background(), 1),
			wantCode: http.StatusBadRequest,
			wantErr:  dtoApi.InvalidFileFormat,
		},
		{
			name: "service internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().
					UpdateProfileAvatar(gomock.Any(), int64(1), gomock.AssignableToTypeOf(&dto.RequestUpdateAvatar{})).
					Return(nil, errors.New("s3"))
			},
			buildReq: func(t *testing.T) *http.Request {
				return newMultipartAvatarRequest(t, "a.png", testAvatarPNG)
			},
			ctx:      withUserID(context.Background(), 1),
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := tt.buildReq(t)
			req = req.WithContext(tt.ctx)

			h.UpdateUserAvatar(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestPositiveProfileHandler_DeleteUserAvatat(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	del := dto.ResponseDeleteProfile{
		UserId: 1, Login: "a", FirstName: "A", Avatar: strPtr(""),
	}

	tests := []struct {
		prepare func(*fields)
		name    string
	}{
		{
			name: "200",
			prepare: func(f *fields) {
				f.svc.EXPECT().DeleteProfileAvatar(gomock.Any(), int64(1)).Return(&del, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/avatar", nil)
			req = req.WithContext(withUserID(req.Context(), 1))

			h.DeleteUserAvatat(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestNegativeProfileHandler_DeleteUserAvatat(t *testing.T) {
	type fields struct {
		svc *mock.MockProfileServiceInterface
	}

	tests := []struct {
		ctx      context.Context
		prepare  func(*fields)
		name     string
		wantErr  dtoApi.ErrorCode
		wantCode int
	}{
		{
			name:     "unauthorized",
			prepare:  nil,
			ctx:      context.Background(),
			wantCode: http.StatusUnauthorized,
			wantErr:  dtoApi.Unauthorized,
		},
		{
			name: "internal",
			prepare: func(f *fields) {
				f.svc.EXPECT().DeleteProfileAvatar(gomock.Any(), int64(1)).Return(nil, errors.New("e"))
			},
			ctx:      withUserID(context.Background(), 1),
			wantCode: http.StatusInternalServerError,
			wantErr:  dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{svc: mock.NewMockProfileServiceInterface(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			h := &ProfileHandler{profileService: f.svc}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/avatar", nil)
			req = req.WithContext(tt.ctx)

			h.DeleteUserAvatat(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
		})
	}
}

func TestNewProfileHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := mock.NewMockProfileServiceInterface(ctrl)
	h := NewProfileHandler(svc)
	require.NotNil(t, h)
	require.Equal(t, svc, h.profileService)
}
