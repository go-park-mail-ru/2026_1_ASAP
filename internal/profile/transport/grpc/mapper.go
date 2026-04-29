package grpc

import (
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapGetProfileToProto(profileDTO *dto.ResponseGetProfile) *profilev1.ResponseGetProfile {
	if profileDTO == nil {
		return nil
	}

	resp := &profilev1.ResponseGetProfile{
		UserId:    profileDTO.UserId,
		FirstName: profileDTO.FirstName,
		Bio:       ptrStringOrEmpty(profileDTO.Bio),
		BirthDate: ptrStringOrEmpty(profileDTO.BirthDate),
		Avatar:    ptrStringOrEmpty(profileDTO.Avatar),
	}

	if profileDTO.LastName != nil {
		resp.LastName = profileDTO.LastName
	}
	if profileDTO.LastSeen != nil {
		resp.LastSeen = timestamppb.New(*profileDTO.LastSeen)
	}

	return resp
}

func mapUpdateProfileToProto(profileDTO *dto.ResponseUpdateProfile) *profilev1.ResponseGetProfile {
	if profileDTO == nil {
		return nil
	}

	resp := &profilev1.ResponseGetProfile{
		UserId:    profileDTO.UserId,
		FirstName: profileDTO.FirstName,
		Bio:       ptrStringOrEmpty(profileDTO.Bio),
		BirthDate: ptrStringOrEmpty(profileDTO.BirthDate),
		Avatar:    ptrStringOrEmpty(profileDTO.Avatar),
	}

	if profileDTO.LastName != nil {
		resp.LastName = profileDTO.LastName
	}
	if profileDTO.LastSeen != nil {
		resp.LastSeen = timestamppb.New(*profileDTO.LastSeen)
	}

	return resp
}

func mapDeleteProfileToProto(profileDTO *dto.ResponseDeleteProfile) *profilev1.ResponseGetProfile {
	if profileDTO == nil {
		return nil
	}

	resp := &profilev1.ResponseGetProfile{
		UserId:    profileDTO.UserId,
		FirstName: profileDTO.FirstName,
		Bio:       ptrStringOrEmpty(profileDTO.Bio),
		BirthDate: ptrStringOrEmpty(profileDTO.BirthDate),
		Avatar:    ptrStringOrEmpty(profileDTO.Avatar),
	}

	if profileDTO.LastName != nil {
		resp.LastName = profileDTO.LastName
	}
	if profileDTO.LastSeen != nil {
		resp.LastSeen = timestamppb.New(*profileDTO.LastSeen)
	}

	return resp
}

func ptrStringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func mapSearchIdByLoginToProto(d *dto.ResponseSearchIdByLogin) *profilev1.ResponseSearchIdByLogin {
	if d == nil {
		return nil
	}
	return &profilev1.ResponseSearchIdByLogin{
		UserId: d.UserId,
		Login:  d.Login,
	}
}
