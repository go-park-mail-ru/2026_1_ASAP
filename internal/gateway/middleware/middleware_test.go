package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func okHandler(t *testing.T, wantUserID int64) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantUserID != 0 {
			require.Equal(t, wantUserID, r.Context().Value(UserID))
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := RequestIDFromContext(r.Context())
		require.True(t, ok)
		require.NotEmpty(t, id)
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	_, ok := RequestIDFromContext(context.Background())
	require.False(t, ok)
}

func TestAccessMiddleware(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "with request id", ctx: context.WithValue(context.Background(), requestID, "req-1")},
		{name: "without request id", ctx: context.Background()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil).WithContext(tt.ctx)
			w := httptest.NewRecorder()

			AccessMiddleware(zap.NewNop())(okHandler(t, 0)).ServeHTTP(w, req)
			require.Equal(t, http.StatusNoContent, w.Code)
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		prepare func(*mock.MockAuthClient)
		name    string
		cookie  string
		want    int
	}{
		{
			name:   "valid session",
			cookie: "sid",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					ValidateSession(gomock.Any(), &authv1.RequestValidateSession{SessionId: "sid"}).
					Return(&authv1.ResponseValidateSession{UserId: 77}, nil)
			},
			want: http.StatusNoContent,
		},
		{
			name: "missing cookie",
			want: http.StatusUnauthorized,
		},
		{
			name:   "session expired",
			cookie: "sid",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					ValidateSession(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED), "expired"))
			},
			want: http.StatusUnauthorized,
		},
		{
			name:   "internal error",
			cookie: "sid",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					ValidateSession(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "failed"))
			},
			want: http.StatusInternalServerError,
		},
		{
			name:   "nil session",
			cookie: "sid",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			want: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := mock.NewMockAuthClient(ctrl)
			if tt.prepare != nil {
				tt.prepare(client)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tt.cookie})
			}
			w := httptest.NewRecorder()

			AuthMiddleware(client)(okHandler(t, 77)).ServeHTTP(w, req)
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestCSRFMiddleware(t *testing.T) {
	tests := []struct {
		prepare func(*mock.MockAuthClient)
		name    string
		cookie  string
		header  string
		want    int
	}{
		{
			name: "missing csrf header",
			want: http.StatusForbidden,
		},
		{
			name:   "missing cookie",
			header: "csrf",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "valid token",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), &authv1.RequestGetCSRFToken{SessionId: "sid"}).
					Return(&authv1.ResponseGetCSRFToken{CsrfToken: "csrf"}, nil)
			},
			want: http.StatusNoContent,
		},
		{
			name:   "mismatch",
			cookie: "sid",
			header: "bad",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().GetCSRFToken(gomock.Any(), gomock.Any()).Return(&authv1.ResponseGetCSRFToken{CsrfToken: "csrf"}, nil)
			},
			want: http.StatusForbidden,
		},
		{
			name:   "token not in session",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.NotFound, int32(authv1.AuthErrorCode_AUTH_ERROR_CSRF_NOT_FOUND), "missing"))
			},
			want: http.StatusForbidden,
		},
		{
			name:   "expired token",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_CSRF_EXPIRED), "expired"))
				c.EXPECT().SetCSRFToken(gomock.Any(), gomock.Any()).Return(&emptypb.Empty{}, nil)
			},
			want: http.StatusForbidden,
		},
		{
			name:   "expired token set error",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_CSRF_EXPIRED), "expired"))
				c.EXPECT().SetCSRFToken(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "failed"))
			},
			want: http.StatusInternalServerError,
		},
		{
			name:   "invalid session",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND), "missing"))
			},
			want: http.StatusUnauthorized,
		},
		{
			name:   "unknown grpc error",
			cookie: "sid",
			header: "csrf",
			prepare: func(c *mock.MockAuthClient) {
				c.EXPECT().
					GetCSRFToken(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "failed"))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := mock.NewMockAuthClient(ctrl)
			if tt.prepare != nil {
				tt.prepare(client)
			}
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set("X-CSRF-TOKEN", tt.header)
			}
			w := httptest.NewRecorder()

			CSRFMiddleware(client)(okHandler(t, 0)).ServeHTTP(w, req)
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestMapGetCSRFGRPCError(t *testing.T) {
	tests := []struct {
		err  error
		want error
		name string
	}{
		{name: "session expired", err: grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED), "expired"), want: errCSRFSessionInvalid},
		{name: "not found", err: grpcerr.New(codes.NotFound, int32(authv1.AuthErrorCode_AUTH_ERROR_CSRF_NOT_FOUND), "missing"), want: errCSRFNotInSession},
		{name: "expired", err: grpcerr.New(codes.Unauthenticated, int32(authv1.AuthErrorCode_AUTH_ERROR_CSRF_EXPIRED), "expired"), want: errCSRFExpiredToken},
		{name: "unknown", err: grpcerr.New(codes.Internal, int32(authv1.AuthErrorCode_AUTH_ERROR_INTERNAL), "failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapGetCSRFGRPCError(tt.err)
			if tt.want == nil {
				require.Equal(t, tt.err, got)
				return
			}
			require.ErrorIs(t, got, tt.want)
		})
	}
}
