package profile

import "time"

type ResponseGetProfile struct {
	UserId   int64      `json:"user_id"`
	Username string     `json:"username"`
	Avatar   *string    `json:"avatar,omitempty"`
	Bio      *string    `json:"bio,omitempty"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}
