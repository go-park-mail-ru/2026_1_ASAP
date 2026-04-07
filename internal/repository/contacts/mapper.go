package contacts

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
)

func toDomainContact(contactModel *ContactModel) *domain.Contact {
	return &domain.Contact{
		UserID: contactModel.UserID,
		FirstName: contactModel.FirstName,
		LastName: null.NullStringToPtrString(contactModel.LastName),
		ContactUserID: contactModel.ContactUserID,
		ContactAvatarUrl: null.NullStringToPtrString(contactModel.ContactAvatarUrl),
		CreatedAt: contactModel.CreatedAt,
		UpdatedAt: contactModel.UpdatedAt,
	}
}

func toModelContact(contactDomain *domain.Contact) *ContactModel {
	return &ContactModel{
		UserID: contactDomain.UserID,
		FirstName: contactDomain.FirstName,
		LastName: null.StringPtrToNullString(contactDomain.LastName),
		ContactUserID: contactDomain.ContactUserID,
		ContactAvatarUrl: null.StringPtrToNullString(contactDomain.ContactAvatarUrl),
		CreatedAt: contactDomain.CreatedAt,
		UpdatedAt: contactDomain.UpdatedAt,
	}
}

/*func toDomainContactUserInfo(contactUserInfoModel *ContactUserInfoModel) *domain.ContactUserInfo {
	return &domain.ContactUserInfo{
		Contact: domain.Contact(contactUserInfoModel.ContactModel),
		Username: contactUserInfoModel.Username,
		Email: contactUserInfoModel.Email,
		AvatarUrl: null.NullStringToPtrString(contactUserInfoModel.AvatarUrl),
		Bio: null.NullStringToPtrString(contactUserInfoModel.Bio),
		LastSeen: null.NullTimeToPtrTime(contactUserInfoModel.LastSeen),
	}
}*/