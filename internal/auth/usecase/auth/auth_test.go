package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"
	dtoVK "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/vkid"

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
		wantID     int64
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
			wantID:  0,
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
			wantID:  0,
			wantErr: domain.ErrEmailAlreadyExists,
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(nil, errors.New("db down"))
			},
			wantID:     0,
			args:       args{ctx: context.Background(), request: req},
			wantAnyErr: true,
			wantSubstr: "failed to create profile",
		},
		{
			name: "Profile create fails after user created",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().
					Create(context.Background(), gomock.AssignableToTypeOf(&domain.User{})).
					Return(&domain.User{ID: 10, Login: req.Login, Email: req.Email}, nil)
			},
			wantID:  10,
			args:    args{ctx: context.Background(), request: req},
			wantErr: nil,
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
			require.Equal(t, tt.wantID, result)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				if tt.wantErr == nil {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tt.wantErr.Error())
				}
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

func TestNegativeAuthService_GetUserPublic(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		wantErr    string
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(gomock.Any(), int64(999)).Return(nil, domain.ErrNotFound)
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			wantErr: "failed to get user by id: user not found",
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByID(gomock.Any(), int64(100)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx:    context.Background(),
				userID: 100,
			},
			wantAnyErr: true,
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
				ProfileService: nil,
			}

			result, err := s.GetUserPublic(tt.args.ctx, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestNegativeAuthService_AuthWithVKID(t *testing.T) {
	type fields struct {
		userRepository *mock.MockUserRepository
		sessionService *mock.MockSessionService
		profileService *mock.MockProfileService
	}

	type args struct {
		ctx     context.Context
		request *dtoVK.RequestAuth
	}

	tests := []struct {
		wantErr    string
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "GetUserByVKID returns unexpected error",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByVKID(gomock.Any(), int64(12345)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				request: &dtoVK.RequestAuth{
					VKUserID:  12345,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john@example.com",
				},
			},
			wantAnyErr: true,
		},
		{
			name: "CreateUserByVKID fails",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByVKID(gomock.Any(), int64(12345)).Return(nil, domain.ErrNotFound)
				f.userRepository.EXPECT().CreateUserByVKID(gomock.Any(), int64(12345), gomock.Any()).Return(nil, errors.New("create failed"))
			},
			args: args{
				ctx: context.Background(),
				request: &dtoVK.RequestAuth{
					VKUserID:  12345,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john@example.com",
				},
			},
			wantAnyErr: true,
		},
		{
			name: "Create profile fails for new user",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByVKID(gomock.Any(), int64(12345)).Return(nil, domain.ErrNotFound)
				f.userRepository.EXPECT().CreateUserByVKID(gomock.Any(), int64(12345), gomock.Any()).DoAndReturn(
					func(_ context.Context, vkID int64, user *domain.User) (*domain.User, error) {
						user.ID = 100
						return user, nil
					})

				f.profileService.EXPECT().Create(gomock.Any(), int64(100), "John").Return(errors.New("profile create failed"))

			},
			args: args{
				ctx: context.Background(),
				request: &dtoVK.RequestAuth{
					VKUserID:  12345,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john@example.com",
				},
			},
			wantAnyErr: true,
		},
		{
			name: "Create session fails for new user",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByVKID(gomock.Any(), int64(12345)).Return(nil, domain.ErrNotFound)
				f.userRepository.EXPECT().CreateUserByVKID(gomock.Any(), int64(12345), gomock.Any()).DoAndReturn(
					func(_ context.Context, vkID int64, user *domain.User) (*domain.User, error) {
						user.ID = 100
						return user, nil
					})

				f.profileService.EXPECT().Create(gomock.Any(), int64(100), "John").Return(nil)
				f.profileService.EXPECT().UpdateName(gomock.Any(), int64(100), "John", "Doe").Return(nil)
				f.profileService.EXPECT().UpdateAvatarFromURL(gomock.Any(), int64(100), "https://avatar.url").Return(nil)

				f.sessionService.EXPECT().CreateSession(gomock.Any(), int64(100)).Return(nil, errors.New("session failed"))
			},
			args: args{
				ctx: context.Background(),
				request: &dtoVK.RequestAuth{
					VKUserID:  12345,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john@example.com",
					AvatarURL: "https://avatar.url",
				},
			},
			wantAnyErr: true,
		},
		{
			name: "Create session fails for existing user",
			prepare: func(f *fields) {
				f.userRepository.EXPECT().GetUserByVKID(gomock.Any(), int64(12345)).Return(&domain.User{
					ID:    200,
					Login: "vk_12345",
					Email: "existing@example.com",
				}, nil)

				f.profileService.EXPECT().Create(gomock.Any(), int64(200), "John").Return(nil)
				f.profileService.EXPECT().UpdateName(gomock.Any(), int64(200), "John", "Doe").Return(nil)
				f.profileService.EXPECT().UpdateAvatarFromURL(gomock.Any(), int64(200), "https://avatar.url").Return(nil)

				f.sessionService.EXPECT().CreateSession(gomock.Any(), int64(200)).Return(nil, errors.New("session failed"))
			},
			args: args{
				ctx: context.Background(),
				request: &dtoVK.RequestAuth{
					VKUserID:  12345,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john@example.com",
					AvatarURL: "https://avatar.url",
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				userRepository: mock.NewMockUserRepository(ctrl),
				sessionService: mock.NewMockSessionService(ctrl),
				profileService: mock.NewMockProfileService(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthService{
				userRepository: f.userRepository,
				SessionService: f.sessionService,
				ProfileService: f.profileService,
			}

			result, err := s.AuthWithVKID(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
