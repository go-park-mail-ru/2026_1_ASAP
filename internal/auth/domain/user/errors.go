package user

import "errors"

var (
	ErrNotFound           = errors.New("profile not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
)
