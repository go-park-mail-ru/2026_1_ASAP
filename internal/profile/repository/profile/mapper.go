package profile

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
)

func toDomainProfile(profileModel *ProfileModel) *domain.Profile {
	return &domain.Profile{
		UserId:    profileModel.UserId,
		FirstName: profileModel.FirstName,
		LastName:  null.NullStringToPtrString(profileModel.LastName),
		Avatar:    null.NullStringToPtrString(profileModel.Avatar),
		Bio:       null.NullStringToPtrString(profileModel.Bio),
		BirthDate: null.NullTimeToPtrTime(profileModel.BirthDate),
		LastSeen:  null.NullTimeToPtrTime(profileModel.LastSeen),
	}
}
