package sessions

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
)

func toModel(session *domain.Session) *SessionModel {
	return &SessionModel{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}

func toDomain(session *SessionModel) *domain.Session {
	return &domain.Session{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}
