package dto

import "time"

type ResponsePayment struct {
	ID               int64
	PaymentID        string
	UserID           int64
	Status           string
	Amount           int32
	SubscriptionDays int32
	PaymentURL       *string
	Message          *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
