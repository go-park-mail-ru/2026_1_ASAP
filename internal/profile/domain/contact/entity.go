package contacts

import "time"

type Contact struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastName         *string
	ContactAvatarUrl *string
	FirstName        string
	UserID           int64
	ContactUserID    int64
}
