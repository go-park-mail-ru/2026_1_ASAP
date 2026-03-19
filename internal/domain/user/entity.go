package user

import "github.com/google/uuid"

type User struct {
	Login        string
	Email        string
	PasswordHash string
	Id           uuid.UUID
}
