package contacts

import "time"

type AddContactRequest struct {
	LastName      *string `json:"last_name,omitempty"`
	FirstName     string  `json:"first_name"`
	ContactUserID int64   `json:"contact_user_id"`
}

type DeleteContactRequest struct {
	ContactUserID int64 `json:"contact_user_id"`
}

type ContactResponse struct {
	CreatedAt        time.Time `json:"created_at"`
	LastName         *string   `json:"last_name,omitempty"`
	ContactAvatarUrl *string   `json:"contact_avatar_url"`
	FirstName        string    `json:"first_name"`
	UserID           int64     `json:"user_id"`
	ContactUserID    int64     `json:"contact_user_id"`
}

type ResponseDeleteSuccess struct {
	ContactUserID int64 `json:"contact_user_id"`
}
