package profile

import "time"

type Profile struct {
	UserId    int64
	Login     string
	FirstName string
	LastName  *string
	Avatar    *string
	Bio       *string
	BirthDate *time.Time
	LastSeen  *time.Time
}
