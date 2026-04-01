package contacts

import (
	"database/sql"
	"time"
)

type ContactModel struct {
	UserID int64 `json:"user_id"`
	ContactName string `json:"contact_name"`
	ContactUserID int64 `json:"contact_user_id"`
	CreatedAt time.Time `json:"created_at"`
	ContactAvatarUrl sql.NullString `json:"contact_avatar_url"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContactUserInfoModel struct {
	ContactModel 
	Username string `json:"username"`
	Email string `json:"email"`
	AvatarUrl sql.NullString `json:"avatar_url"`
	Bio sql.NullString `json:"bio"`
	LastSeen sql.NullTime `json:"last_seen"`
}