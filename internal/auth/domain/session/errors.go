package session

import "errors"

var (
	ErrNotFound = errors.New("session not found")
	ErrExpired  = errors.New("session expired")

	ErrCSRFNotMatch = errors.New("CSRF not match")
	ErrCSRFNotFound = errors.New("CSRF not found")
	ErrCSRFExpired  = errors.New("CSRF expired")
)
