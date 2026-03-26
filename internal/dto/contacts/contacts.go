package contacts

import "time"

type AddContactRequest struct {
	ContactUserID int64 `json:"contact_user_id"`
	ContactName string `json:"contact_name"`
}

type DeleteContactRequest struct {
	ContactUserID int64 `json:"contact_user_id"`
}

type ContactResponse struct {
	UserID int64 `json:"user_id"` 
	ContactUserID int64 `json:"contact_user_id"`
	ContactName string `json:"contact_name"`
	CreatedAt time.Time `json:"created_at"`
}

type ResponseDeleteSuccess struct {
	ContactUserID int64 `json:"contact_user_id"`
}