package contacts

import "time"

type Contact struct {
	UserID int64 
	FirstName string  
	LastName  *string 
	ContactUserID int64 
	ContactAvatarUrl *string 
	CreatedAt time.Time
	UpdatedAt time.Time 
}

type ContactUserInfo struct {
	Contact 
	Username string `json:"username"`
	Email string `json:"email"`
	AvatarUrl *string `json:"avatar_url"`
	Bio *string `json:"bio"`
	LastSeen *time.Time `json:"last_seen"`
}