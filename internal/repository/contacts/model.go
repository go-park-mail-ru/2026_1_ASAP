package contacts

import (
	"database/sql"
	"time"
)

type ContactModel struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FirstName        string
	LastName         sql.NullString
	ContactAvatarUrl sql.NullString
	UserID           int64
	ContactUserID    int64
}

type ContactUserInfoModel struct {
	LastSeen  sql.NullTime   `json:"last_seen"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	AvatarUrl sql.NullString `json:"avatar_url"`
	Bio       sql.NullString `json:"bio"`
	ContactModel
}
