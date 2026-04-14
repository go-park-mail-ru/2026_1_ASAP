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

type ContactUserInfo struct {
	AvatarUrl *string    `json:"avatar_url"`
	Bio       *string    `json:"bio"`
	LastSeen  *time.Time `json:"last_seen"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Contact
}
