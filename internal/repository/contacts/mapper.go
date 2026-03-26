package contacts

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/null"
)

func toDomainContact(contactModel *ContactModel) *domain.Contact {
	return &domain.Contact{
		UserID: contactModel.UserID,
		ContactName: contactModel.ContactName,
		ContactUserID: contactModel.ContactUserID,
		CreatedAt: contactModel.CreatedAt,
		UpdatedAt: contactModel.UpdatedAt,
	}
}

func toModelContact(contactDomain *domain.Contact) *ContactModel {
	return &ContactModel{
		UserID: contactDomain.UserID,
		ContactName: contactDomain.ContactName,
		ContactUserID: contactDomain.ContactUserID,
		CreatedAt: contactDomain.CreatedAt,
		UpdatedAt: contactDomain.UpdatedAt,
	}
}

func toDomainContactUserInfo(contactUserInfoModel *ContactUserInfoModel) *domain.ContactUserInfo {
	return &domain.ContactUserInfo{
		Contact: domain.Contact(contactUserInfoModel.ContactModel),
		Username: contactUserInfoModel.Username,
		Email: contactUserInfoModel.Email,
		AvatarUrl: null.NullStringToPtrString(contactUserInfoModel.AvatarUrl),
		Bio: null.NullStringToPtrString(contactUserInfoModel.Bio),
		LastSeen: null.NullTimeToPtrTime(contactUserInfoModel.LastSeen),
	}
}