package payment

type CreatePaymentRequest struct {
	Amount           int32  `json:"amount"`
	SubscriptionDays int32  `json:"subscription_days"`
	Description      string `json:"description,omitempty"`
}
