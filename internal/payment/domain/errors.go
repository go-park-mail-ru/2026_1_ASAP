package domain

import "errors"

var (
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrDuplicatePayment      = errors.New("payment already exists")
	ErrInvalidPaymentRequest = errors.New("invalid payment request")
	ErrPaymentReturnURLUnset = errors.New("payment return url is not configured")
)
