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
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthUsecaseInterface interface {
	Register(ctx context.Context, request *dtoAuth.RequestRegistrate) (int64, error)
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

	logger *zap.Logger
}

func NewServer(authUsecase AuthUsecaseInterface, sessionUsecase SessionUsecaseInterface, logger *zap.Logger) *AuthServer {
	return &AuthServer{
		authUsecase:    authUsecase,
		sessionUsecase: sessionUsecase,
		logger:         logger,
	}
}

func authErr(code codes.Code, appCode authv1.AuthErrorCode, msg string) error {
	return grpcerr.New(code, int32(appCode), msg)
}

func (a *AuthServer) Login(ctx context.Context, req *authv1.RequestLogin) (*authv1.ResponseLogin, error) {
	if req == nil || req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "login and password are required")
	}

	sessionData, err := a.authUsecase.Login(ctx, &dtoAuth.RequestLogin{
		Login:    req.GetLogin(),
		Password: req.GetPassword(),
	})
	if err != nil {
		if errors.Is(err, domainUser.ErrNotFound) || errors.Is(err, domainUser.ErrInvalidCredentials) {
			return nil, authErr(codes.Unauthenticated, authv1.AuthErrorCode_AUTH_ERROR_INVALID_CREDENTIALS, "invalid credentials")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to login")
	}

	return &authv1.ResponseLogin{
		Login: req.GetLogin(),
		Session: &authv1.SessionMeta{
			SessionId: sessionData.SessionID,
			CsrfToken: sessionData.CSRFToken,
			ExpiresAt: timestamppb.New(sessionData.Expire),
		},
	}, nil
}

func (a *AuthServer) Register(ctx context.Context, req *authv1.RequestRegister) (*authv1.ResponseRegister, error) {
	if req == nil || req.GetLogin() == "" || req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "login, email and password are required")
	}

	userID, err := a.authUsecase.Register(ctx, &dtoAuth.RequestRegistrate{
		Login:    req.GetLogin(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrInvalidInput):
			a.Log(ctx).Info("failed to register user", zap.Error(err))
			return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, err.Error())
		case errors.Is(err, domainUser.ErrLoginAlreadyExists):
			a.Log(ctx).Info("failed to register user", zap.Error(err))
			return nil, authErr(codes.AlreadyExists, authv1.AuthErrorCode_AUTH_ERROR_LOGIN_ALREADY_EXISTS, "login already exists")
		case errors.Is(err, domainUser.ErrEmailAlreadyExists):
			a.Log(ctx).Info("failed to register user", zap.Error(err))
			return nil, authErr(codes.AlreadyExists, authv1.AuthErrorCode_AUTH_ERROR_EMAIL_ALREADY_EXISTS, "email already exists")
		}
		a.Log(ctx).Error("failed to register user", zap.Error(err))
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to register")
	}

	return &authv1.ResponseRegister{
		Login:  req.GetLogin(),
		Email:  req.GetEmail(),
		UserId: userID,
	}, nil
}

func (a *AuthServer) Logout(ctx context.Context, req *authv1.RequestLogout) (*authv1.ResponseLogout, error) {
	if req == nil || req.GetSessionId() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "session_id is required")
	}

	err := a.authUsecase.Logout(ctx, &dtoAuth.RequestLogout{SessionID: req.GetSessionId()})
	if err != nil {
		if errors.Is(err, domainSession.ErrNotFound) {
			return nil, authErr(codes.NotFound, authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND, "session not found")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to logout")
	}

	return &authv1.ResponseLogout{}, nil
}

func (a *AuthServer) ValidateSession(ctx context.Context, req *authv1.RequestValidateSession) (*authv1.ResponseValidateSession, error) {
	if req == nil || req.GetSessionId() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "session_id is required")
	}

	uid, err := a.sessionUsecase.GetUserID(ctx, req.GetSessionId())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, authErr(codes.NotFound, authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, authErr(codes.FailedPrecondition, authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED, "session expired")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to validate session")
	}

	return &authv1.ResponseValidateSession{UserId: uid}, nil
}

func (a *AuthServer) GetCSRFToken(ctx context.Context, req *authv1.RequestGetCSRFToken) (*authv1.ResponseGetCSRFToken, error) {
	if req == nil || req.GetSessionId() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "session_id is required")
	}

	token, err := a.sessionUsecase.GetCSRFToken(ctx, req.GetSessionId())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, authErr(codes.NotFound, authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, authErr(codes.FailedPrecondition, authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED, "session expired")
		case errors.Is(err, domainSession.ErrCSRFNotFound):
			return nil, authErr(codes.FailedPrecondition, authv1.AuthErrorCode_AUTH_ERROR_CSRF_NOT_FOUND, "csrf token not found")
		case errors.Is(err, domainSession.ErrCSRFExpired):
			return nil, authErr(codes.FailedPrecondition, authv1.AuthErrorCode_AUTH_ERROR_CSRF_EXPIRED, "csrf token expired")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to get csrf token")
	}

	return &authv1.ResponseGetCSRFToken{CsrfToken: token}, nil
}

func (a *AuthServer) SetCSRFToken(ctx context.Context, req *authv1.RequestSetCSRFToken) (*emptypb.Empty, error) {
	if req == nil || req.GetSessionId() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "session_id is required")
	}
	if req.GetCsrfToken() == "" {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "csrf_token is required")
	}

	err := a.sessionUsecase.SetCSRFToken(ctx, req.GetSessionId(), req.GetCsrfToken())
	if err != nil {
		switch {
		case errors.Is(err, domainSession.ErrNotFound):
			return nil, authErr(codes.NotFound, authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND, "session not found")
		case errors.Is(err, domainSession.ErrExpired):
			return nil, authErr(codes.FailedPrecondition, authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED, "session expired")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to set csrf token")
	}

	return &emptypb.Empty{}, nil
}

func (a *AuthServer) GetUserPublic(ctx context.Context, req *authv1.RequestGetUserPublic) (*authv1.ResponseGetUserPublic, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, authErr(codes.InvalidArgument, authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT, "user_id is required")
	}

	u, err := a.authUsecase.GetUserPublic(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, domainUser.ErrNotFound) {
			return nil, authErr(codes.NotFound, authv1.AuthErrorCode_AUTH_ERROR_USER_NOT_FOUND, "user not found")
		}
		return nil, authErr(codes.Internal, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "failed to get user")
	}

	return &authv1.ResponseGetUserPublic{
		Login: u.Login,
		Email: u.Email,
	}, nil
}

func (a *AuthServer) AuthVKID(context.Context, *authv1.RequestVKID) (*authv1.ResponseLogin, error) {
	return nil, authErr(codes.Unimplemented, authv1.AuthErrorCode_AUTH_ERROR_INTERNAL, "AuthVKID not implemented on auth service")
}

func (a *AuthServer) Log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, a.logger)
}
