package grpc

import (
	"bytes"
	"context"
	"errors"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProfileServiceInterface interface {
	GetUserProfile(ctx context.Context, userID int64) (response *profile.ResponseGetProfile, err error)
	UpdateProfileBio(ctx context.Context, userID int64, request *profile.RequestUpdateBio) (response *profile.ResponseUpdateProfile, err error)
	UpdateProfileAvatar(ctx context.Context, userID int64, request *profile.RequestUpdateAvatar) (response *profile.ResponseUpdateProfile, err error)
	UpdateProfileBirthDate(ctx context.Context, userID int64, request *profile.RequestUpdateBirthDate) (response *profile.ResponseUpdateProfile, err error)
	SearchIdByLogin(ctx context.Context, login *profile.RequestSearchIdByLogin) (response *profile.ResponseSearchIdByLogin, err error)
	UpdateProfileName(ctx context.Context, userID int64, request *profile.RequestUpdateName) (*profile.ResponseUpdateProfile, error)
	DeleteProfileAvatar(ctx context.Context, userID int64) (response *profile.ResponseDeleteProfile, err error)
}

type ProfileServer struct {
	profilev1.UnimplementedProfileServer
	profileUseCase ProfileServiceInterface
}

func NewProfileServer(profileUseCase ProfileServiceInterface) *ProfileServer {
	return &ProfileServer{
		profileUseCase: profileUseCase,
	}
}

func (p ProfileServer) GetProfile(ctx context.Context, request *profilev1.RequestGetProfile) (*profilev1.ResponseGetProfile, error) {
	if request == nil || request.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	profileDTO, err := p.profileUseCase.GetUserProfile(ctx, request.GetUserId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Error(codes.Internal, "profile internal error")
	}

	return mapGetProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileAvatar(ctx context.Context, avatar *profilev1.RequestUpdateAvatar) (*profilev1.ResponseGetProfile, error) {
	if avatar == nil || avatar.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(avatar.GetContent()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "avatar content is required")
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
		case errors.Is(err, domain.ErrAvatarTooLarge),
			errors.Is(err, domain.ErrInvalidAvatarType),
			errors.Is(err, domain.ErrEmptyAvatar):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.NotFound, "profile not found")
		default:
			return nil, status.Error(codes.Internal, "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileBio(ctx context.Context, req *profilev1.RequestUpdateBio) (*profilev1.ResponseGetProfile, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.NotFound, "profile not found")
		default:
			return nil, status.Error(codes.Internal, "profile internal error")
		}
	}
	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileBirthDate(ctx context.Context, date *profilev1.RequestUpdateBirthDate) (*profilev1.ResponseGetProfile, error) {
	if date == nil || date.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
		case errors.Is(err, domain.ErrInvalidBirthDate), errors.Is(err, domain.ErrInvalidBirthDateFormat):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.NotFound, "profile not found")
		default:
			return nil, status.Error(codes.Internal, "profile internal error")
		}
	}

	return mapUpdateProfileToProto(profileDTO), nil
}

func (p ProfileServer) UpdateProfileName(ctx context.Context, name *profilev1.RequestUpdateName) (*profilev1.ResponseGetProfile, error) {
	if name == nil || name.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
		case errors.Is(err, domain.ErrEmptyFirstName):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.NotFound, "profile not found")
		default:
			return nil, status.Error(codes.Internal, "profile internal error")
		}
	}

	return mapUpdateProfileToProto(profileDTO), nil
}
