package messages

import (
	"database/sql"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
)

type AttachmentModel struct {
	CreatedAt        time.Time
	FileURL          sql.NullString
	FileName         sql.NullString
	MimeType         sql.NullString
	ContactAvatarURL sql.NullString
	ContactFirstName sql.NullString
	ContactLastName  sql.NullString
	Type             string
	Id               int64
	MessageId        int64
	ContactUserID    sql.NullInt64
	FileSize         sql.NullInt64
	SortOrder        int
}

func attachmentToDomain(m *AttachmentModel) domain.MessageAttachment {
	if m == nil {
		return domain.MessageAttachment{}
	}
	out := domain.MessageAttachment{
		Id:        m.Id,
		MessageId: m.MessageId,
		Type:      domain.AttachmentType(m.Type),
		SortOrder: m.SortOrder,
		CreatedAt: m.CreatedAt,
	}
	if m.FileURL.Valid {
		v := m.FileURL.String
		out.FileURL = &v
	}
	if m.FileName.Valid {
		v := m.FileName.String
		out.FileName = &v
	}
	if m.MimeType.Valid {
		v := m.MimeType.String
		out.MimeType = &v
	}
	if m.FileSize.Valid {
		v := m.FileSize.Int64
		out.FileSize = &v
	}
	if m.ContactUserID.Valid {
		v := m.ContactUserID.Int64
		out.ContactUserID = &v
	}
	if m.ContactFirstName.Valid {
		v := m.ContactFirstName.String
		out.ContactFirstName = &v
	}
	if m.ContactLastName.Valid {
		v := m.ContactLastName.String
		out.ContactLastName = &v
	}
	if m.ContactAvatarURL.Valid {
		v := m.ContactAvatarURL.String
		out.ContactAvatarURL = &v
	}
	return out
}

func attachmentFromDomain(a domain.MessageAttachment) AttachmentModel {
	m := AttachmentModel{
		MessageId: a.MessageId,
		Type:      string(a.Type),
		SortOrder: a.SortOrder,
	}
	if a.FileURL != nil {
		m.FileURL = sql.NullString{String: *a.FileURL, Valid: true}
	}
	if a.FileName != nil {
		m.FileName = sql.NullString{String: *a.FileName, Valid: true}
	}
	if a.MimeType != nil {
		m.MimeType = sql.NullString{String: *a.MimeType, Valid: true}
	}
	if a.FileSize != nil {
		m.FileSize = sql.NullInt64{Int64: *a.FileSize, Valid: true}
	}
	if a.ContactUserID != nil {
		m.ContactUserID = sql.NullInt64{Int64: *a.ContactUserID, Valid: true}
	}
	if a.ContactFirstName != nil {
		m.ContactFirstName = sql.NullString{String: *a.ContactFirstName, Valid: true}
	}
	if a.ContactLastName != nil {
		m.ContactLastName = sql.NullString{String: *a.ContactLastName, Valid: true}
	}
	if a.ContactAvatarURL != nil {
		m.ContactAvatarURL = sql.NullString{String: *a.ContactAvatarURL, Valid: true}
	}
	return m
}
