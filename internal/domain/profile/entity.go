package profile

import "time"

type Profile struct {
	UserId   int64
	Username string
	Avatar   *string
	Bio      *string
	LastSeen *time.Time
}
