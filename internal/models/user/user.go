package models

import "github.com/google/uuid"

type User struct {
	Id           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
}
