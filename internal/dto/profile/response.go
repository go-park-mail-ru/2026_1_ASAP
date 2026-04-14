package profile

import "time"

type ResponseGetProfile struct {
	BirthDate *string    `json:"birth_date,omitempty"`
	LastName  *string    `json:"last_name,omitempty"`
	Avatar    *string    `json:"avatar,omitempty"`
	Bio       *string    `json:"bio,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Login     string     `json:"login"`
	FirstName string     `json:"first_name"`
	Email     string     `json:"email"`
	UserId    int64      `json:"user_id"`
}

type ResponseUpdateProfile struct {
	LastName  *string    `json:"last_name,omitempty"`
	BirthDate *string    `json:"birth_date,omitempty"`
	Avatar    *string    `json:"avatar,omitempty"`
	Bio       *string    `json:"bio,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Login     string     `json:"login"`
	FirstName string     `json:"first_name"`
	UserId    int64      `json:"user_id"`
}

type ResponseSearchIdByLogin struct {
	Login  string `json:"login"`
	UserId int64  `json:"user_id"`
}

type ResponseDeleteProfile struct {
	LastName  *string    `json:"last_name,omitempty"`
	BirthDate *string    `json:"birth_date,omitempty"`
	Avatar    *string    `json:"avatar"`
	Bio       *string    `json:"bio,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Login     string     `json:"login"`
	FirstName string     `json:"first_name"`
	UserId    int64      `json:"user_id"`
}
