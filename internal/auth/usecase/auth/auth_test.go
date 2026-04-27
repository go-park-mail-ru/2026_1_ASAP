package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/usecase/auth/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/hash"
)

func TestPositiveAuthService_Register(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestRegistrate
	}

	tests := []struct {
		args    args
		prepare func(*fields)
		wantID  int64
		name    string
	}{
		{
			name: "Registers user and returns user_id",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					DoAndReturn(func(_ context.Context, u *domain.User) (*domain.User, error) {
						require.Equal(t, "newuser", u.Login)
						require.Equal(t, "newuser@mail.test", u.Email)
						require.NotEmpty(t, u.PasswordHash)
						return &domain.User{
							ID:    55,
							Login: u.Login,
							Email: u.Email,
						}, nil
					})
			},
			args: args{
				ctx: context.Background(),
				request: &dtoAuth.RequestRegistrate{
					Login:    "newuser",
					Email:    "newuser@mail.test",
					Password: "SecurePassw0rd!",
				},
			},
			wantID: 55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			result, err := s.Register(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.wantID, result)
		})
	}
}

func TestNegativeAuthService_Register(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestRegistrate
	}

	req := &dtoAuth.RequestRegistrate{
		Login:    "user1",
		Email:    "user1@mail.test",
		Password: "SecurePassw0rd!",
	}

	tests := []struct {
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
		wantSubstr string
		wantAnyErr bool
	}{
		{
			name: "Login already exists",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(nil, domain.ErrLoginAlreadyExists)
			},
			args:    args{ctx: context.Background(), request: req},
			wantErr: domain.ErrLoginAlreadyExists,
		},
		{
			name: "Email already exists",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(nil, domain.ErrEmailAlreadyExists)
			},
			args:    args{ctx: context.Background(), request: req},
			wantErr: domain.ErrEmailAlreadyExists,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(nil, errors.New("db down"))
			},
			args:       args{ctx: context.Background(), request: req},
			wantAnyErr: true,
			wantSubstr: "failed to create profile",
		},
		{
			name: "CreateSession fails after profile created",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(&domain.User{ID: 10, Login: req.Login, Email: req.Email}, nil)
				f.sessionService.EXPECT().
					CreateSession(context.Background(), int64(10)).
					Return(nil, errors.New("redis down"))
			},
			args:       args{ctx: context.Background(), request: req},
			wantAnyErr: true,
			wantSubstr: "failed to create session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			result, err := s.Register(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveAuthService_Login(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestLogin
	}

	pass := "CorrectHorseBattery99!"
	hashStr, err := hash.HashPassword(pass)
	require.NoError(t, err)

	sessionExpire := time.Date(2030, 7, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		args    args
		prepare func(*fields)
		want    *dtoSession.SessionDTO
		name    string
	}{
		{
			name: "Valid credentials",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					GetUserByLogin(context.Background(), "alice").
					Return(&domain.User{
						ID:           3,
						Login:        "alice",
						PasswordHash: hashStr,
					}, nil)
				f.sessionService.EXPECT().
					CreateSession(context.Background(), int64(3)).
					Return(&dtoSession.SessionDTO{
						SessionID: "sid-alice",
						CSRFToken: "csrf",
						Expire:    sessionExpire,
					}, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dtoAuth.RequestLogin{
					Login:    "alice",
					Password: pass,
				},
			},
			want: &dtoSession.SessionDTO{
				SessionID: "sid-alice",
				CSRFToken: "csrf",
				Expire:    sessionExpire,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			result, err := s.Login(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeAuthService_Login(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestLogin
	}

	pass := "SamePassword123!"
	hashStr, err := hash.HashPassword(pass)
	require.NoError(t, err)

	tests := []struct {
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
		wantSubstr string
		wantAnyErr bool
	}{
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					GetUserByLogin(context.Background(), "nobody").
					Return(nil, domain.ErrNotFound)
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogin{Login: "nobody", Password: pass},
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "Invalid password",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					GetUserByLogin(context.Background(), "bob").
					Return(&domain.User{
						ID:           4,
						Login:        "bob",
						PasswordHash: hashStr,
					}, nil)
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogin{Login: "bob", Password: "wrong-password"},
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name: "GetUserByLogin error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					GetUserByLogin(context.Background(), "bob").
					Return(nil, errors.New("timeout"))
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogin{Login: "bob", Password: pass},
			},
			wantAnyErr: true,
			wantSubstr: "failed login",
		},
		{
			name: "CreateSession error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					GetUserByLogin(context.Background(), "bob").
					Return(&domain.User{
						ID:           4,
						Login:        "bob",
						PasswordHash: hashStr,
					}, nil)
				f.sessionService.EXPECT().
					CreateSession(context.Background(), int64(4)).
					Return(nil, errors.New("session failed"))
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogin{Login: "bob", Password: pass},
			},
			wantAnyErr: true,
			wantSubstr: "failed to create session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			result, err := s.Login(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveAuthService_Logout(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestLogout
	}

	tests := []struct {
		args    args
		prepare func(*fields)
		name    string
	}{
		{
			name: "Deletes session",
			prepare: func(f *fields) {
				f.sessionService.EXPECT().
					DeleteSession(context.Background(), "sess-to-clear").
					Return(nil)
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogout{SessionID: "sess-to-clear"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			err := s.Logout(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
		})
	}
}

func TestNegativeAuthService_Logout(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx     context.Context
		request *dtoAuth.RequestLogout
	}

	sid := "missing-sid"

	tests := []struct {
		args       args
		wantErr    error
		prepare    func(*fields)
		name       string
		wantSubstr string
		wantAnyErr bool
	}{
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionService.EXPECT().
					DeleteSession(context.Background(), sid).
					Return(domainSession.ErrNotFound)
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogout{SessionID: sid},
			},
			wantErr: domainSession.ErrNotFound,
		},
		{
			name: "DeleteSession error",
			prepare: func(f *fields) {
				f.sessionService.EXPECT().
					DeleteSession(context.Background(), sid).
					Return(errors.New("redis err"))
			},
			args: args{
				ctx:     context.Background(),
				request: &dtoAuth.RequestLogout{SessionID: sid},
			},
			wantAnyErr: true,
			wantSubstr: "failed to logout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
			}
			err := s.Logout(tt.args.ctx, tt.args.request)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
