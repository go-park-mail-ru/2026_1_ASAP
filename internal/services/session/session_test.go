package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session/mock"
)

const testSessionTTL = 2 * time.Hour

func TestPositiveSessionService_CreateSession(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		createdSession    *domain.Session
		sessionTTL        time.Duration
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "Creates session with CSRF and expiry",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().
					CreateSession(context.Background(), gomock.AssignableToTypeOf(&domain.Session{})).
					DoAndReturn(func(_ context.Context, s *domain.Session) (string, error) {
						f.createdSession = s
						require.Equal(t, int64(1001), s.UserID)
						require.NotEmpty(t, s.SessionID)
						require.NotEmpty(t, s.CSRFToken)
						require.True(t, s.ExpiresAt.After(time.Now()))
						require.True(t, s.CSRFExpiresAt.After(time.Now()))
						return s.SessionID, nil
					})
			},
			args: args{ctx: context.Background(), userID: 1001},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        testSessionTTL,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.CreateSession(tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, f.createdSession)
			require.Equal(t, f.createdSession.SessionID, result.SessionID)
			require.Equal(t, f.createdSession.CSRFToken, result.CSRFToken)
			require.Equal(t, f.createdSession.ExpiresAt, result.Expire)
		})
	}
}

func TestNegativeSessionService_CreateSession(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantAnyErr bool
	}{
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().
					CreateSession(context.Background(), gomock.AssignableToTypeOf(&domain.Session{})).
					Return("", errors.New("redis unavailable"))
			},
			args:       args{ctx: context.Background(), userID: 1},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        testSessionTTL,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.CreateSession(tt.args.ctx, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create session")
			}
		})
	}
}

func TestPositiveSessionService_GetUserID(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	expiresOK := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    int64
	}{
		{
			name: "Valid session",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "session-1").Return(&domain.Session{
					SessionID: "session-1",
					UserID:    42,
					ExpiresAt: expiresOK,
				}, nil)
			},
			args: args{ctx: context.Background(), sessionID: "session-1"},
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        24 * time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.GetUserID(tt.args.ctx, tt.args.sessionID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeSessionService_GetUserID(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		args       args
		name       string
		wantAnyErr bool
	}{
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "session-1").Return(nil, domain.ErrNotFound)
			},
			args:    args{ctx: context.Background(), sessionID: "session-1"},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "session-1").Return(&domain.Session{
					SessionID: "session-1",
					UserID:    42,
					ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			args:    args{ctx: context.Background(), sessionID: "session-1"},
			wantErr: domain.ErrExpired,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "session-1").Return(nil, errors.New("conn reset"))
			},
			args:       args{ctx: context.Background(), sessionID: "session-1"},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        24 * time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.GetUserID(tt.args.ctx, tt.args.sessionID)
			require.Zero(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get session")
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveSessionService_GetCSRFToken(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    string
	}{
		{
			name: "Returns CSRF token",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid-csrf").Return(&domain.Session{
					SessionID:     "sid-csrf",
					ExpiresAt:     time.Now().Add(time.Hour),
					CSRFToken:     "csrf-secret",
					CSRFExpiresAt: time.Now().Add(30 * time.Minute),
				}, nil)
			},
			args: args{ctx: context.Background(), sessionID: "sid-csrf"},
			want: "csrf-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.GetCSRFToken(tt.args.ctx, tt.args.sessionID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeSessionService_GetCSRFToken(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		args       args
		name       string
		wantAnyErr bool
	}{
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid").Return(nil, domain.ErrNotFound)
			},
			args:    args{ctx: context.Background(), sessionID: "sid"},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid").Return(&domain.Session{
					SessionID: "sid",
					ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			args:    args{ctx: context.Background(), sessionID: "sid"},
			wantErr: domain.ErrExpired,
		},
		{
			name: "CSRF not set",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid").Return(&domain.Session{
					SessionID:     "sid",
					ExpiresAt:     time.Now().Add(time.Hour),
					CSRFToken:     "",
					CSRFExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			args:    args{ctx: context.Background(), sessionID: "sid"},
			wantErr: domain.ErrCSRFNotFound,
		},
		{
			name: "CSRF expired",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid").Return(&domain.Session{
					SessionID:     "sid",
					ExpiresAt:     time.Now().Add(time.Hour),
					CSRFToken:     "tok",
					CSRFExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			args:    args{ctx: context.Background(), sessionID: "sid"},
			wantErr: domain.ErrCSRFExpired,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), "sid").Return(nil, errors.New("timeout"))
			},
			args:       args{ctx: context.Background(), sessionID: "sid"},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			result, err := s.GetCSRFToken(tt.args.ctx, tt.args.sessionID)
			require.Empty(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get session")
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveSessionService_SetCSRFToken(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
		token     string
	}

	sid := "session-set"

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "Updates CSRF and persists session",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), sid).Return(&domain.Session{
					SessionID: sid,
					UserID:    7,
					ExpiresAt: time.Now().Add(time.Hour),
					CSRFToken: "old",
				}, nil)
				f.sessionRepository.EXPECT().
					CreateSession(context.Background(), gomock.AssignableToTypeOf(&domain.Session{})).
					DoAndReturn(func(_ context.Context, s *domain.Session) (string, error) {
						require.Equal(t, "new-csrf", s.CSRFToken)
						require.Equal(t, sid, s.SessionID)
						require.Equal(t, int64(7), s.UserID)
						require.True(t, s.CSRFExpiresAt.After(time.Now()))
						return sid, nil
					})
			},
			args: args{ctx: context.Background(), sessionID: sid, token: "new-csrf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			err := s.SetCSRFToken(tt.args.ctx, tt.args.sessionID, tt.args.token)
			require.NoError(t, err)
		})
	}
}

func TestNegativeSessionService_SetCSRFToken(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
		token     string
	}

	sid := "session-set"

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		args       args
		name       string
		wantAnyErr bool
	}{
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), sid).Return(nil, domain.ErrNotFound)
			},
			args:    args{ctx: context.Background(), sessionID: sid, token: "t"},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), sid).Return(&domain.Session{
					SessionID: sid,
					ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			args:    args{ctx: context.Background(), sessionID: sid, token: "t"},
			wantErr: domain.ErrExpired,
		},
		{
			name: "CreateSession fails",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().GetSession(context.Background(), sid).Return(&domain.Session{
					SessionID: sid,
					UserID:    1,
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
				f.sessionRepository.EXPECT().
					CreateSession(context.Background(), gomock.AssignableToTypeOf(&domain.Session{})).
					Return("", errors.New("write failed"))
			},
			args:       args{ctx: context.Background(), sessionID: sid, token: "tok"},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			err := s.SetCSRFToken(tt.args.ctx, tt.args.sessionID, tt.args.token)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create session")
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveSessionService_DeleteSession(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	sid := "to-delete"

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "Deletes session",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().DeleteSession(context.Background(), sid).Return(nil)
			},
			args: args{ctx: context.Background(), sessionID: sid},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			err := s.DeleteSession(tt.args.ctx, tt.args.sessionID)
			require.NoError(t, err)
		})
	}
}

func TestNegativeSessionService_DeleteSession(t *testing.T) {
	type fields struct {
		sessionRepository *mock.MockSessionRepository
		sessionTTL        time.Duration
	}

	type args struct {
		ctx       context.Context
		sessionID string
	}

	sid := "to-delete"

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		args       args
		name       string
		wantAnyErr bool
	}{
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().DeleteSession(context.Background(), sid).Return(domain.ErrNotFound)
			},
			args:    args{ctx: context.Background(), sessionID: sid},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Unknown error",
			prepare: func(f *fields) {
				f.sessionRepository.EXPECT().DeleteSession(context.Background(), sid).Return(errors.New("redis err"))
			},
			args:       args{ctx: context.Background(), sessionID: sid},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				sessionRepository: mock.NewMockSessionRepository(ctrl),
				sessionTTL:        time.Hour,
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SessionService{
				sessionRepository: f.sessionRepository,
				sessionTTL:        f.sessionTTL,
			}
			err := s.DeleteSession(tt.args.ctx, tt.args.sessionID)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete session")
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
