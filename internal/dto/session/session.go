package session

import "time"

type SessionDTO struct {
	SessionID string
	Expire    time.Time
}
