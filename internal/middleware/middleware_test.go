package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func decodeAPIError(t *testing.T, rr *httptest.ResponseRecorder) dtoApi.ApiErrorResponse {
	t.Helper()
	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	return got
}

func TestPositiveMiddleware_AuthMiddleware(t *testing.T) {
	type fields struct {
		session *mock.MockSessionService
	}

	tests := []struct {
		name    string
		prepare func(*fields)
	}{
		{
			name: "sets UserID and SessionID in context",
			prepare: func(f *fields) {
				f.session.EXPECT().
					GetUserID(gomock.Any(), "sid-7").
					Return(int64(7), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{session: mock.NewMockSessionService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			var gotUID int64
			var gotSID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUID, _ = r.Context().Value(UserID).(int64)
				gotSID, _ = r.Context().Value(SessionID).(string)
				w.WriteHeader(http.StatusTeapot)
			})

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid-7"})

			AuthMiddleware(f.session)(next).ServeHTTP(rr, req)

			require.Equal(t, http.StatusTeapot, rr.Code)
			require.Equal(t, int64(7), gotUID)
			require.Equal(t, "sid-7", gotSID)
		})
	}
}

func TestNegativeMiddleware_AuthMiddleware(t *testing.T) {
	type fields struct {
		session *mock.MockSessionService
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		addCookie  bool
		cookieVal  string
		wantCode   int
		wantErr    dtoApi.ErrorCode
		nextCalled bool
	}{
		{
			name:       "no session cookie",
			prepare:    nil,
			addCookie:  false,
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.Unauthorized,
			nextCalled: false,
		},
		{
			name: "session expired",
			prepare: func(f *fields) {
				f.session.EXPECT().GetUserID(gomock.Any(), "exp").Return(int64(0), domainSession.ErrExpired)
			},
			addCookie:  true,
			cookieVal:  "exp",
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.Unauthorized,
			nextCalled: false,
		},
		{
			name: "session not found",
			prepare: func(f *fields) {
				f.session.EXPECT().GetUserID(gomock.Any(), "gone").Return(int64(0), domainSession.ErrNotFound)
			},
			addCookie:  true,
			cookieVal:  "gone",
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.Unauthorized,
			nextCalled: false,
		},
		{
			name: "GetUserID other error",
			prepare: func(f *fields) {
				f.session.EXPECT().GetUserID(gomock.Any(), "x").Return(int64(0), errors.New("redis"))
			},
			addCookie:  true,
			cookieVal:  "x",
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.InternalError,
			nextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{session: mock.NewMockSessionService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.addCookie {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tt.cookieVal})
			}

			AuthMiddleware(f.session)(next).ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
			require.Equal(t, tt.nextCalled, nextCalled)
		})
	}
}

func TestPositiveMiddleware_CSRFMiddleware(t *testing.T) {
	type fields struct {
		csrfSvc *mock.MockCSRFTokenService
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		token   string
	}{
		{
			name: "calls next when token matches",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("tok-abc", nil)
			},
			token: "tok-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{csrfSvc: mock.NewMockCSRFTokenService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("X-CSRF-TOKEN", tt.token)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})

			CSRFMiddleware(f.csrfSvc)(next).ServeHTTP(rr, req)

			require.True(t, nextCalled)
			require.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestNegativeMiddleware_CSRFMiddleware(t *testing.T) {
	type fields struct {
		csrfSvc *mock.MockCSRFTokenService
	}

	tests := []struct {
		name        string
		prepare     func(*fields)
		setHeader   bool
		headerVal   string
		addCookie   bool
		cookieVal   string
		wantCode    int
		wantErr     dtoApi.ErrorCode
		wantNewCSRF bool
		nextCalled  bool
	}{
		{
			name:       "missing CSRF header",
			prepare:    nil,
			setHeader:  false,
			addCookie:  true,
			cookieVal:  "sid",
			wantCode:   http.StatusForbidden,
			wantErr:    dtoApi.CSRFTokenMissing,
			nextCalled: false,
		},
		{
			name:       "missing session cookie",
			prepare:    nil,
			setHeader:  true,
			headerVal:  "tok",
			addCookie:  false,
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.Unauthorized,
			nextCalled: false,
		},
		{
			name: "CSRF not in session",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("", domainSession.ErrCSRFNotFound)
			},
			setHeader:  true,
			headerVal:  "any",
			addCookie:  true,
			cookieVal:  "sid",
			wantCode:   http.StatusForbidden,
			wantErr:    dtoApi.CSRFTokenNotInSession,
			nextCalled: false,
		},
		{
			name: "CSRF expired rotates token",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("", domainSession.ErrCSRFExpired)
				f.csrfSvc.EXPECT().
					SetCSRFToken(gomock.Any(), "sid", gomock.Any()).
					Return(nil)
			},
			setHeader:   true,
			headerVal:   "old",
			addCookie:   true,
			cookieVal:   "sid",
			wantCode:    http.StatusForbidden,
			wantErr:     dtoApi.CSRFTokenExpired,
			wantNewCSRF: true,
			nextCalled:  false,
		},
		{
			name: "session not found from GetCSRFToken",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("", domainSession.ErrNotFound)
			},
			setHeader:  true,
			headerVal:  "tok",
			addCookie:  true,
			cookieVal:  "sid",
			wantCode:   http.StatusUnauthorized,
			wantErr:    dtoApi.Unauthorized,
			nextCalled: false,
		},
		{
			name: "GetCSRFToken internal error",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("", errors.New("db"))
			},
			setHeader:  true,
			headerVal:  "tok",
			addCookie:  true,
			cookieVal:  "sid",
			wantCode:   http.StatusInternalServerError,
			wantErr:    dtoApi.InternalError,
			nextCalled: false,
		},
		{
			name: "token mismatch",
			prepare: func(f *fields) {
				f.csrfSvc.EXPECT().GetCSRFToken(gomock.Any(), "sid").Return("server-token", nil)
			},
			setHeader:  true,
			headerVal:  "client-token",
			addCookie:  true,
			cookieVal:  "sid",
			wantCode:   http.StatusForbidden,
			wantErr:    dtoApi.CSRFTokenMismatch,
			nextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{csrfSvc: mock.NewMockCSRFTokenService(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			})

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.setHeader {
				req.Header.Set("X-CSRF-TOKEN", tt.headerVal)
			}
			if tt.addCookie {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tt.cookieVal})
			}

			CSRFMiddleware(f.csrfSvc)(next).ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)
			got := decodeAPIError(t, rr)
			require.Equal(t, tt.wantErr, got.Errors[0].Code)
			require.Equal(t, tt.nextCalled, nextCalled)
			if tt.wantNewCSRF {
				require.NotEmpty(t, rr.Header().Get("X-NEW-CSRF-TOKEN"))
			}
		})
	}
}

func TestPositiveMiddleware_RequestIDMiddleware(t *testing.T) {
	type fields struct{}

	tests := []struct {
		name    string
		prepare func(*fields)
	}{
		{name: "injects UUID into context", prepare: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			var rid string
			var ok bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rid, ok = RequestIDFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			RequestIDMiddleware()(next).ServeHTTP(rr, req)

			require.True(t, ok)
			require.Len(t, rid, 36)
			require.Equal(t, http.StatusNoContent, rr.Code)
		})
	}
}

func TestPositiveMiddleware_AccessMiddleware(t *testing.T) {
	type fields struct{}

	tests := []struct {
		name       string
		prepare    func(*fields)
		withReqID  bool
		wantReqLog string
	}{
		{
			name:       "logs unknown_request_id without RequestID middleware",
			prepare:    nil,
			withReqID:  false,
			wantReqLog: "unknown_request_id",
		},
		{
			name:      "logs request id from context",
			prepare:   nil,
			withReqID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			core, recorded := observer.New(zap.InfoLevel)
			logger := zap.New(core)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})

			handler := AccessMiddleware(logger)(next)
			if tt.withReqID {
				handler = RequestIDMiddleware()(handler)
			}

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/path/here", nil)
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusTeapot, rr.Code)
			entries := recorded.All()
			require.Len(t, entries, 1)
			require.Equal(t, "access", entries[0].Message)

			var reqID string
			for _, c := range entries[0].Context {
				if c.Key == "request_id" {
					reqID = c.String
					break
				}
			}
			if tt.withReqID {
				require.Len(t, reqID, 36)
			} else {
				require.Equal(t, tt.wantReqLog, reqID)
			}
		})
	}
}
