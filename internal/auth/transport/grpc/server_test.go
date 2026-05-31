//go:generate mockgen -destination=mock/auth_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc AuthUsecaseInterface
//go:generate mockgen -destination=mock/session_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc SessionUsecaseInterface
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc/mock"
)

func TestPositiveAuthServer_Login(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestLogin
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseLogin
		name    string
		args    args
	}{
		{
			name: "Successful login",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Login(gomock.Any(), &dtoAuth.RequestLogin{
					Login:    "testuser",
					Password: "password123",
				}).Return(&dtoSession.SessionDTO{
					SessionID: "session123",
					CSRFToken: "csrf123",
					Expire:    now,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "testuser",
					Password: "password123",
				},
			},
			want: &authv1.ResponseLogin{
				Login: "testuser",
				Session: &authv1.SessionMeta{
					SessionId: "session123",
					CsrfToken: "csrf123",
					ExpiresAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.Login(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetLogin(), resp.GetLogin())
			require.Equal(t, tt.want.GetSession().GetSessionId(), resp.GetSession().GetSessionId())
		})
	}
}

func TestNegativeAuthServer_Login(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestLogin
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty login",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "",
					Password: "password",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty password",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "testuser",
					Password: "",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid credentials - user not found",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, domainUser.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "wrong",
					Password: "wrong",
				},
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "Invalid credentials - wrong password",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, domainUser.ErrInvalidCredentials)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "testuser",
					Password: "wrong",
				},
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogin{
					Login:    "testuser",
					Password: "password",
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.Login(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_Register(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestRegister
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseRegister
		name    string
		args    args
	}{
		{
			name: "Successful register",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), &dtoAuth.RequestRegistrate{
					Login:    "newuser",
					Email:    "new@example.com",
					Password: "password123",
				}).Return(int64(100), nil)

				f.sessionUsecase.EXPECT().CreateSession(gomock.Any(), int64(100)).Return(&dtoSession.SessionDTO{
					SessionID: "session123",
					CSRFToken: "csrf123",
					Expire:    now,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "newuser",
					Email:    "new@example.com",
					Password: "password123",
				},
			},
			want: &authv1.ResponseRegister{
				Login:  "newuser",
				Email:  "new@example.com",
				UserId: 100,
				Session: &authv1.SessionMeta{
					SessionId: "session123",
					CsrfToken: "csrf123",
					ExpiresAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.Register(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetLogin(), resp.GetLogin())
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
		})
	}
}

func TestNegativeAuthServer_Register(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestRegister
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty login",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "",
					Email:    "test@example.com",
					Password: "password",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty email",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "testuser",
					Email:    "",
					Password: "password",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty password",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "testuser",
					Email:    "test@example.com",
					Password: "",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Login already exists",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), gomock.Any()).Return(int64(0), domainUser.ErrLoginAlreadyExists)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "existing",
					Email:    "new@example.com",
					Password: "password",
				},
			},
			wantCode: codes.AlreadyExists,
		},
		{
			name: "Email already exists",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), gomock.Any()).Return(int64(0), domainUser.ErrEmailAlreadyExists)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "newuser",
					Email:    "existing@example.com",
					Password: "password",
				},
			},
			wantCode: codes.AlreadyExists,
		},
		{
			name: "Invalid input",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), gomock.Any()).Return(int64(0), domainUser.ErrInvalidInput)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "invalid!",
					Email:    "test@example.com",
					Password: "password",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "newuser",
					Email:    "new@example.com",
					Password: "password",
				},
			},
			wantCode: codes.Internal,
		},
		{
			name: "Session creation error",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Register(gomock.Any(), gomock.Any()).Return(int64(100), nil)
				f.sessionUsecase.EXPECT().CreateSession(gomock.Any(), int64(100)).Return(nil, errors.New("session error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestRegister{
					Login:    "newuser",
					Email:    "new@example.com",
					Password: "password",
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.Register(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_Logout(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestLogout
	}

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseLogout
		name    string
		args    args
	}{
		{
			name: "Successful logout",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Logout(gomock.Any(), &dtoAuth.RequestLogout{
					SessionID: "session123",
				}).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogout{SessionId: "session123"},
			},
			want: &authv1.ResponseLogout{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.Logout(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestNegativeAuthServer_Logout(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestLogout
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty session_id",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogout{SessionId: ""},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(domainSession.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogout{SessionId: "invalid"},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestLogout{SessionId: "session123"},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.Logout(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_ValidateSession(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestValidateSession
	}

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseValidateSession
		name    string
		args    args
	}{
		{
			name: "Successful validate session",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetUserID(gomock.Any(), "session123").Return(int64(100), nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestValidateSession{SessionId: "session123"},
			},
			want: &authv1.ResponseValidateSession{UserId: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.ValidateSession(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
		})
	}
}

func TestNegativeAuthServer_ValidateSession(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestValidateSession
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty session_id",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestValidateSession{SessionId: ""},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetUserID(gomock.Any(), "invalid").Return(int64(0), domainSession.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestValidateSession{SessionId: "invalid"},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetUserID(gomock.Any(), "expired").Return(int64(0), domainSession.ErrExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestValidateSession{SessionId: "expired"},
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetUserID(gomock.Any(), "session123").Return(int64(0), errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestValidateSession{SessionId: "session123"},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.ValidateSession(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_GetCSRFToken(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestGetCSRFToken
	}

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseGetCSRFToken
		name    string
		args    args
	}{
		{
			name: "Successful get CSRF token",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "session123").Return("csrf_token_123", nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "session123"},
			},
			want: &authv1.ResponseGetCSRFToken{CsrfToken: "csrf_token_123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.GetCSRFToken(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetCsrfToken(), resp.GetCsrfToken())
		})
	}
}

func TestNegativeAuthServer_GetCSRFToken(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestGetCSRFToken
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty session_id",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: ""},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "invalid").Return("", domainSession.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "invalid"},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "expired").Return("", domainSession.ErrExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "expired"},
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "CSRF not found",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "session123").Return("", domainSession.ErrCSRFNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "session123"},
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "CSRF expired",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "session123").Return("", domainSession.ErrCSRFExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "session123"},
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().GetCSRFToken(gomock.Any(), "session123").Return("", errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetCSRFToken{SessionId: "session123"},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.GetCSRFToken(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_SetCSRFToken(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestSetCSRFToken
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful set CSRF token",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().SetCSRFToken(gomock.Any(), "session123", "csrf_token_123").Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "session123",
					CsrfToken: "csrf_token_123",
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.SetCSRFToken(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestNegativeAuthServer_SetCSRFToken(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestSetCSRFToken
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty session_id",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "",
					CsrfToken: "token",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty csrf_token",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "session123",
					CsrfToken: "",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Session not found",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().SetCSRFToken(gomock.Any(), "invalid", "token").Return(domainSession.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "invalid",
					CsrfToken: "token",
				},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Session expired",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().SetCSRFToken(gomock.Any(), "expired", "token").Return(domainSession.ErrExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "expired",
					CsrfToken: "token",
				},
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.sessionUsecase.EXPECT().SetCSRFToken(gomock.Any(), "session123", "token").Return(errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestSetCSRFToken{
					SessionId: "session123",
					CsrfToken: "token",
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.SetCSRFToken(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveAuthServer_GetUserPublic(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestGetUserPublic
	}

	tests := []struct {
		prepare func(*fields)
		want    *authv1.ResponseGetUserPublic
		name    string
		args    args
	}{
		{
			name: "Successful get user public",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().GetUserPublic(gomock.Any(), int64(100)).Return(&dtoAuth.ResponseUserPublic{
					Login: "testuser",
					Email: "test@example.com",
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetUserPublic{UserId: 100},
			},
			want: &authv1.ResponseGetUserPublic{
				Login: "testuser",
				Email: "test@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			resp, err := s.GetUserPublic(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetLogin(), resp.GetLogin())
			require.Equal(t, tt.want.GetEmail(), resp.GetEmail())
		})
	}
}

func TestNegativeAuthServer_GetUserPublic(t *testing.T) {
	type fields struct {
		authUsecase    *mock.MockAuthUsecaseInterface
		sessionUsecase *mock.MockSessionUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *authv1.RequestGetUserPublic
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id (zero)",
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetUserPublic{UserId: 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().GetUserPublic(gomock.Any(), int64(999)).Return(nil, domainUser.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetUserPublic{UserId: 999},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Usecase returns other error",
			prepare: func(f *fields) {
				f.authUsecase.EXPECT().GetUserPublic(gomock.Any(), int64(100)).Return(nil, errors.New("some error"))
			},
			args: args{
				ctx: context.Background(),
				req: &authv1.RequestGetUserPublic{UserId: 100},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				authUsecase:    mock.NewMockAuthUsecaseInterface(ctrl),
				sessionUsecase: mock.NewMockSessionUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &AuthServer{
				authUsecase:    f.authUsecase,
				sessionUsecase: f.sessionUsecase,
				vkidConfig:     config.VKIDConfig{},
				logger:         zap.NewNop(),
			}

			_, err := s.GetUserPublic(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}
