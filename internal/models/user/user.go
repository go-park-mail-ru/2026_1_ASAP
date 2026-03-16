package models

import "github.com/google/uuid"

type User struct {
	Login        string    `json:"login"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Id           uuid.UUID `json:"id"`
}
