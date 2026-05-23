package grpc

import (
	"bytes"
	"context"
	"errors"
	"strings"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ContactServiceInterface interface {
	GetContacts(ctx context.Context, userID int64) ([]*dto.ContactResponse, error)
	HasContact(ctx context.Context, userID, contactUserID int64) (bool, error)
	GetContact(ctx context.Context, userID, contactUserID int64) (*dto.ContactResponse, error)
	AddContact(ctx context.Context, contactRequest dto.AddContactRequest, userID int64) (*dto.ContactResponse, error)
	DeleteContact(ctx context.Context, contactRequest dto.DeleteContactRequest, userID int64) error
}

type ProfileServiceInterface interface {
	CreateProfile(ctx context.Context, profile *profile.RequestCreateProfile) error
	GetUserProfile(ctx context.Context, userID int64) (response *profile.ResponseGetProfile, err error)
	UpdateProfileBio(ctx context.Context, userID int64, request *profile.RequestUpdateBio) (response *profile.ResponseUpdateProfile, err error)
	UpdateProfileAvatar(ctx context.Context, userID int64, request *profile.RequestUpdateAvatar) (response *profile.ResponseUpdateProfile, err error)
	UpdateProfileAvatarURL(ctx context.Context, userID int64, request *profile.RequestUpdateAvatarURL) (response *profile.ResponseUpdateProfile, err error)
	UpdateProfileBirthDate(ctx context.Context, userID int64, request *profile.RequestUpdateBirthDate) (response *profile.ResponseUpdateProfile, err error)
	SearchIdByLogin(ctx context.Context, login *profile.RequestSearchIdByLogin) (response *profile.ResponseSearchIdByLogin, err error)
	UpdateProfileName(ctx context.Context, userID int64, request *profile.RequestUpdateName) (*profile.ResponseUpdateProfile, error)
	DeleteProfileAvatar(ctx context.Context, userID int64) (response *profile.ResponseDeleteProfile, err error)
	UpdateLastSeen(ctx context.Context, userID int64) error
}

type ProfileServer struct {
	profilev1.UnimplementedProfileServer
	profileUseCase ProfileServiceInterface
	contactService ContactServiceInterface

	logger *zap.Logger
}

func (p ProfileServer) CreateProfile(ctx context.Context, createProfile *profilev1.RequestCreateProfile) (*emptypb.Empty, error) {
	err := p.profileUseCase.CreateProfile(ctx, &profile.RequestCreateProfile{
		ProfileID: createProfile.GetProfileId(),
		FirstName: createProfile.GetFirstName(),
	})
	if err != nil {
		p.Log(ctx).Error("failed to create profile", zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
	}

	return &emptypb.Empty{}, nil
}

func NewProfileServer(profileUseCase ProfileServiceInterface, contactService ContactServiceInterface, logger *zap.Logger) *ProfileServer {
	return &ProfileServer{
		profileUseCase: profileUseCase,
		contactService: contactService,

		logger: logger,
	}
}

func (p ProfileServer) GetProfile(ctx context.Context, request *profilev1.RequestGetProfile) (*profilev1.ResponseGetProfile, error) {
	if request == nil || request.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}
	profileDTO, err := p.profileUseCase.GetUserProfile(ctx, request.GetUserId())
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", request.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		}
		p.Log(ctx).Error("failed to get profile", zap.Int64("user_id", request.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
	}
	return mapGetProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileAvatar(ctx context.Context, avatar *profilev1.RequestUpdateAvatar) (*profilev1.ResponseGetProfile, error) {
	if avatar == nil || avatar.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}
	if len(avatar.GetContent()) == 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_EMPTY), "avatar content is required")
	}
	profileDTO, err := p.profileUseCase.UpdateProfileAvatar(ctx, avatar.GetUserId(), &profile.RequestUpdateAvatar{
		File: &media.FileInput{
			Body:        bytes.NewReader(avatar.GetContent()),
			ContentType: avatar.GetType(),
			Size:        int64(len(avatar.GetContent())),
		},
	})
	if err != nil {
		switch {
		case errors.Is(err, pdomain.ErrAvatarTooLarge):
			p.Log(ctx).Info("avatar too large", zap.Int64("user_id", avatar.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_TOO_LARGE), "avatar too large")
		case errors.Is(err, pdomain.ErrInvalidAvatarType):
			p.Log(ctx).Info("avatar invalid type", zap.Int64("user_id", avatar.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_INVALID_TYPE), "avatar invalid type")
		case errors.Is(err, pdomain.ErrEmptyAvatar):
			p.Log(ctx).Info("avatar is empty", zap.Int64("user_id", avatar.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_EMPTY), "avatar is empty")
		case errors.Is(err, pdomain.ErrNotFound):
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", avatar.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		default:
			p.Log(ctx).Error("failed to update profile avatar", zap.Int64("user_id", avatar.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileAvatarURL(ctx context.Context, req *profilev1.RequestUpdateAvatarURL) (*profilev1.ResponseGetProfile, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}
	if strings.TrimSpace(req.GetAvatarUrl()) == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_EMPTY), "avatar_url is required")
	}

	profileDTO, err := p.profileUseCase.UpdateProfileAvatarURL(ctx, req.GetUserId(), &profile.RequestUpdateAvatarURL{
		AvatarURL: strings.TrimSpace(req.GetAvatarUrl()),
	})
	if err != nil {
		switch {
		case errors.Is(err, pdomain.ErrEmptyAvatar):
			p.Log(ctx).Info("avatar url is empty", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_AVATAR_EMPTY), "avatar_url is empty")
		case errors.Is(err, pdomain.ErrNotFound):
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		default:
			p.Log(ctx).Error("failed to update profile avatar url", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileBio(ctx context.Context, req *profilev1.RequestUpdateBio) (*profilev1.ResponseGetProfile, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}

	var bioPtr *string
	if req.Bio != nil {
		v := req.GetBio()
		bioPtr = &v
	}
	profileDTO, err := p.profileUseCase.UpdateProfileBio(ctx, req.GetUserId(), &profile.RequestUpdateBio{
		Bio: bioPtr,
	})
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		}
		p.Log(ctx).Error("failed to update profile bio", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileBirthDate(ctx context.Context, date *profilev1.RequestUpdateBirthDate) (*profilev1.ResponseGetProfile, error) {
	if date == nil || date.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}

	var birthDatePtr *string
	if date.BirthDate != nil {
		v := date.GetBirthDate()
		birthDatePtr = &v
	}

	profileDTO, err := p.profileUseCase.UpdateProfileBirthDate(ctx, date.GetUserId(), &profile.RequestUpdateBirthDate{
		BirthDate: birthDatePtr,
	})
	if err != nil {
		switch {
		case errors.Is(err, pdomain.ErrInvalidBirthDate):
			p.Log(ctx).Info("invalid birth date", zap.Int64("user_id", date.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_BIRTH_DATE_INVALID), "invalid birth date")
		case errors.Is(err, pdomain.ErrInvalidBirthDateFormat):
			p.Log(ctx).Info("invalid birth date format", zap.Int64("user_id", date.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_BIRTH_DATE_FORMAT), "invalid birth date format")
		case errors.Is(err, pdomain.ErrNotFound):
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", date.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		default:
			p.Log(ctx).Error("failed to update profile birth date", zap.Int64("user_id", date.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileName(ctx context.Context, name *profilev1.RequestUpdateName) (*profilev1.ResponseGetProfile, error) {
	if name == nil || name.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}

	var lastNamePtr *string
	if name.SecondName != nil {
		v := name.GetSecondName()
		lastNamePtr = &v
	}

	profileDTO, err := p.profileUseCase.UpdateProfileName(ctx, name.GetUserId(), &profile.RequestUpdateName{
		FirstName: name.GetFirstName(),
		LastName:  lastNamePtr,
	})
	if err != nil {
		switch {
		case errors.Is(err, pdomain.ErrEmptyFirstName):
			p.Log(ctx).Info("first name cannot be empty", zap.Int64("user_id", name.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_FIRST_NAME_EMPTY), "first name cannot be empty")
		case errors.Is(err, pdomain.ErrNotFound):
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", name.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		default:
			p.Log(ctx).Error("failed to update profile name", zap.Int64("user_id", name.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateLastSeen(ctx context.Context, request *profilev1.RequestUpdateLastSeen) (*emptypb.Empty, error) {
	if request == nil || request.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "user_id is required")
	}
	if err := p.profileUseCase.UpdateLastSeen(ctx, request.GetUserId()); err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			p.Log(ctx).Info("profile not found", zap.Int64("user_id", request.GetUserId()), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		}
		p.Log(ctx).Error("failed to update last seen", zap.Int64("user_id", request.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
	}
	return &emptypb.Empty{}, nil
}

func (p ProfileServer) SearchIdByLogin(ctx context.Context, request *profilev1.RequestSearchIdByLogin) (*profilev1.ResponseSearchIdByLogin, error) {
	if request == nil {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "request is required")
	}
	login := strings.TrimSpace(request.GetLogin())
	if login == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INVALID_INPUT), "login is required")
	}
	out, err := p.profileUseCase.SearchIdByLogin(ctx, &profile.RequestSearchIdByLogin{Login: login})
	if err != nil {
		if errors.Is(err, pdomain.ErrNotFound) {
			p.Log(ctx).Info("profile not found by login", zap.String("login", login), zap.Error(err))
			return nil, grpcerr.New(codes.NotFound, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_NOT_FOUND), "profile not found")
		}
		p.Log(ctx).Error("failed to search profile by login", zap.String("login", login), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(profilev1.ProfileErrorCode_PROFILE_ERROR_INTERNAL), "profile internal error")
	}
	return mapSearchIdByLoginToProto(out), nil
}

func (p ProfileServer) Log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, p.logger)
}
