package domain

import "errors"

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionExpired  = errors.New("subscription expired")
)
