package contacts

import (
	contactdom "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

func toDomainContact(contactModel *ContactModel) *contactdom.Contact {
	return &contactdom.Contact{
		UserID:           contactModel.UserID,
		FirstName:        contactModel.FirstName,
		LastName:         null.NullStringToPtrString(contactModel.LastName),
		ContactUserID:    contactModel.ContactUserID,
		ContactAvatarUrl: null.NullStringToPtrString(contactModel.ContactAvatarUrl),
		CreatedAt:        contactModel.CreatedAt,
		UpdatedAt:        contactModel.UpdatedAt,
	}
}

func toModelContact(contactDomain *contactdom.Contact) *ContactModel {
	return &ContactModel{
		UserID:           contactDomain.UserID,
		FirstName:        contactDomain.FirstName,
		LastName:         null.StringPtrToNullString(contactDomain.LastName),
		ContactUserID:    contactDomain.ContactUserID,
		ContactAvatarUrl: null.StringPtrToNullString(contactDomain.ContactAvatarUrl),
		CreatedAt:        contactDomain.CreatedAt,
		UpdatedAt:        contactDomain.UpdatedAt,
	}
}
