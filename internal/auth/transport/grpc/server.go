package grpc

import (
	"context"
	"errors"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/session"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/domain/user"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/session"
	dtoVK "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/vkid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthUsecaseInterface interface {
	Register(ctx context.Context, request *dtoAuth.RequestRegistrate) (*dtoSession.SessionDTO, error)
	Login(ctx context.Context, request *dtoAuth.RequestLogin) (*dtoSession.SessionDTO, error)
	Logout(ctx context.Context, request *dtoAuth.RequestLogout) error
	AuthWithVKID(ctx context.Context, request *dtoVK.RequestAuth) (*dtoSession.SessionDTO, error)
	GetUserPublic(ctx context.Context, userID int64) (*dtoAuth.ResponseUserPublic, error)
}

type SessionUsecaseInterface interface {
	GetUserID(ctx context.Context, sessionID string) (int64, error)
	GetCSRFToken(ctx context.Context, sessionID string) (string, error)
	SetCSRFToken(ctx context.Context, sessionID string, token string) error
}

type AuthServer struct {
	authv1.UnimplementedAuthServer

	authUsecase    AuthUsecaseInterface
	sessionUsecase SessionUsecaseInterface
}

func (a *AuthServer) GetCSRFToken(ctx context.Context, request *authv1.RequestGetCSRFToken) (*authv1.ResponseGetCSRFToken, error) {
	if request == nil || request.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	token, err := a.sessionUsecase.GetCSRFToken(ctx, request.GetSessionId())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, status.Error(codes.NotFound, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, status.Error(codes.FailedPrecondition, "session expired")
		case errors.Is(err, domainSession.ErrCSRFNotFound):
			return nil, status.Error(codes.FailedPrecondition, "csrf token not found")
		case errors.Is(err, domainSession.ErrCSRFExpired):
			return nil, status.Error(codes.FailedPrecondition, "csrf token expired")
		default:
			return nil, status.Error(codes.Internal, "failed to get csrf token")
		}
	}
	return &authv1.ResponseGetCSRFToken{CsrfToken: token}, nil
}

func (a *AuthServer) SetCSRFToken(ctx context.Context, request *authv1.RequestSetCSRFToken) (*emptypb.Empty, error) {
	if request == nil || request.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if request.GetCsrfToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "csrf_token is required")
	}
	err := a.sessionUsecase.SetCSRFToken(ctx, request.GetSessionId(), request.GetCsrfToken())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, status.Error(codes.NotFound, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, status.Error(codes.FailedPrecondition, "session expired")
		default:
			return nil, status.Error(codes.Internal, "failed to set csrf token")
		}
	}
	return &emptypb.Empty{}, nil
}

func (a *AuthServer) Login(ctx context.Context, login *authv1.RequestLogin) (*authv1.ResponseLogin, error) {
	if login == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if login.GetLogin() == "" || login.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password are required")
	}

	sessionData, err := a.authUsecase.Login(ctx, &dtoAuth.RequestLogin{
		Login:    login.GetLogin(),
		Password: login.GetPassword(),
	})
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrNotFound), errors.Is(err, domainUser.ErrInvalidCredentials):
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		default:
			return nil, status.Error(codes.Internal, "failed to login")
		}
	}

	return &authv1.ResponseLogin{
		Login: login.GetLogin(),
		Session: &authv1.SessionMeta{
			SessionId: sessionData.SessionID,
			CsrfToken: sessionData.CSRFToken,
			ExpiresAt: timestamppb.New(sessionData.Expire),
		},
	}, nil
}

func (a *AuthServer) Register(ctx context.Context, register *authv1.RequestRegister) (*authv1.ResponseRegister, error) {
	if register == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if register.GetLogin() == "" || register.GetEmail() == "" || register.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "login, email and password are required")
	}

	_, err := a.authUsecase.Register(ctx, &dtoAuth.RequestRegistrate{
		Login:    register.GetLogin(),
		Email:    register.GetEmail(),
		Password: register.GetPassword(),
	})
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrInvalidInput):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, domainUser.ErrLoginAlreadyExists), errors.Is(err, domainUser.ErrEmailAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		default:
			return nil, status.Error(codes.Internal, "failed to register")
		}
	}

	return &authv1.ResponseRegister{
		Login: register.GetLogin(),
		Email: register.GetEmail(),
	}, nil
}

func NewServer(authUsecase AuthUsecaseInterface, sessionUsecase SessionUsecaseInterface) *AuthServer {
	return &AuthServer{
		authUsecase:    authUsecase,
		sessionUsecase: sessionUsecase,
	}
}

func (a *AuthServer) Logout(ctx context.Context, request *authv1.RequestLogout) (*authv1.ResponseLogout, error) {
	if request == nil || request.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	err := a.authUsecase.Logout(ctx, &dtoAuth.RequestLogout{SessionID: request.GetSessionId()})
	if err != nil {
		if errors.Is(err, domainSession.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Error(codes.Internal, "failed to logout")
	}
	return &authv1.ResponseLogout{}, nil
}

func (a *AuthServer) AuthVKID(context.Context, *authv1.RequestVKID) (*authv1.ResponseLogin, error) {
	return nil, status.Error(codes.Unimplemented, "AuthVKID not implemented on auth service")
}

func (a *AuthServer) ValidateSession(ctx context.Context, req *authv1.RequestValidateSession) (*authv1.ResponseValidateSession, error) {
	if req == nil || req.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	uid, err := a.sessionUsecase.GetUserID(ctx, req.GetSessionId())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, status.Error(codes.NotFound, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, status.Error(codes.FailedPrecondition, "session expired")
		default:
			return nil, status.Error(codes.Internal, "validate session failed")
		}
	}
	return &authv1.ResponseValidateSession{UserId: uid}, nil
}

func (a *AuthServer) GetUserPublic(ctx context.Context, req *authv1.RequestGetUserPublic) (*authv1.ResponseGetUserPublic, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	u, err := a.authUsecase.GetUserPublic(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, domainUser.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}
	return &authv1.ResponseGetUserPublic{
		Login: u.Login,
		Email: u.Email,
	}, nil
}
