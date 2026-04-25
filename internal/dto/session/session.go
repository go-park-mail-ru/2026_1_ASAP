package session

import "time"

type SessionDTO struct {
	Expire    time.Time
	SessionID string
	CSRFToken string
	UserID    string
}
