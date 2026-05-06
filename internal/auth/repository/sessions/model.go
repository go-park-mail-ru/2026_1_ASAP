package sessions

import "time"

type SessionModel struct {
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	CSRFExpiresAt time.Time `json:"csrf_expires_at,omitempty"`
	SessionID     string    `json:"session_id"`
	CSRFToken     string    `json:"csrf_token,omitempty"`
	UserID        int64     `json:"user_id"`
}
