package contacts

import "time"

type AddContactRequest struct {
	ContactUserID int64 `json:"contact_user_id"`
	FirstName string     `json:"first_name"`
	LastName  *string    `json:"last_name,omitempty"`
}

type DeleteContactRequest struct {
	ContactUserID int64 `json:"contact_user_id"`
}

type ContactResponse struct {
	UserID int64 `json:"user_id"` 
	ContactUserID int64 `json:"contact_user_id"`
	FirstName string     `json:"first_name"`
	LastName  *string    `json:"last_name,omitempty"`
	ContactAvatarUrl  *string `json:"contact_avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type ResponseDeleteSuccess struct {
	ContactUserID int64 `json:"contact_user_id"`
}