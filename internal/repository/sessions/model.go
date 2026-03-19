package sessions

import "time"

type SessionModel struct {
	SessionID string    `json:"session_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
