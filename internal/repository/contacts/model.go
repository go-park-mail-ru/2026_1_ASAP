package contacts

import (
	"database/sql"
	"time"
)

type ContactModel struct {
	UserID int64 
	FirstName string  
	LastName  sql.NullString
	ContactUserID int64 
	ContactAvatarUrl sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time 
}

type ContactUserInfoModel struct {
	ContactModel 
	Username string `json:"username"`
	Email string `json:"email"`
	AvatarUrl sql.NullString `json:"avatar_url"`
	Bio sql.NullString `json:"bio"`
	LastSeen sql.NullTime `json:"last_seen"`
}