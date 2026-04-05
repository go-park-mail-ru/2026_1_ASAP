package session

import "errors"

var (
	ErrNotFound = errors.New("session not found")
	ErrExpired  = errors.New("session expired")
)
