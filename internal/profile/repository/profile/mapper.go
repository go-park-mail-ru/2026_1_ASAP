package profile

import (
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
)

func toDomainProfile(profileModel *ProfileModel) *pdomain.Profile {
	return &pdomain.Profile{
		UserId:    profileModel.UserId,
		FirstName: profileModel.FirstName,
		LastName:  null.NullStringToPtrString(profileModel.LastName),
		Avatar:    null.NullStringToPtrString(profileModel.Avatar),
		Bio:       null.NullStringToPtrString(profileModel.Bio),
		BirthDate: null.NullTimeToPtrTime(profileModel.BirthDate),
		LastSeen:  null.NullTimeToPtrTime(profileModel.LastSeen),
	}
}
