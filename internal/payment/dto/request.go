package dto

type RequestCreatePayment struct {
	UserID           int64
	PaymentID        string
	Status           string
	Amount           int32
	SubscriptionDays int32
}

type RequestGetPayment struct {
	ID int64
}
