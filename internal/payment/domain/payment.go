package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusWaitingForCapture PaymentStatus = "waiting_for_capture"
	PaymentStatusSucceeded         PaymentStatus = "succeeded"
	PaymentStatusCanceled          PaymentStatus = "canceled"
)

type Payment struct {
	ID               int64
	PaymentID        string
	UserID           int64
	Status           PaymentStatus
	Amount           int32
	SubscriptionDays int32
	PaymentURL       *string
	Message          *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
