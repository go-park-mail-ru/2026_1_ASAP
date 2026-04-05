package profile

import "time"

type ResponseGetProfile struct {
	UserId    int64      `json:"user_id"`
	Login     string     `json:"login"`
	FirstName string     `json:"first_name"`
	Email     string     `json:"email,omitempty"`
	BirthDate *string    `json:"birth_date,omitempty"`
	LastName  *string    `json:"last_name,omitempty"`
	Avatar    *string    `json:"avatar,omitempty"`
	Bio       *string    `json:"bio,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

type ResponseUpdateProfile struct {
	UserId    int64      `json:"user_id"`
	Login     string     `json:"login"`
	FirstName string     `json:"first_name"`
	LastName  *string    `json:"last_name,omitempty"`
	BirthDate *string    `json:"birth_date,omitempty"`
	Avatar    *string    `json:"avatar,omitempty"`
	Bio       *string    `json:"bio,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

type ResponseSearchIdByLogin struct {
	UserId int64  `json:"user_id"`
	Login  string `json:"login"`
}
