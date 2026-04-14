package profile

import "time"

type Profile struct {
	LastName  *string
	Avatar    *string
	Bio       *string
	BirthDate *time.Time
	LastSeen  *time.Time
	Login     string
	Email     string
	FirstName string
	UserId    int64
}
