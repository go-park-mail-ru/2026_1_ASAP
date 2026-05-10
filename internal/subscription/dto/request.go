package dto

type RequestActivateSubscription struct {
	UserID int64
	Days   int64
}

type RequestCancelSubscription struct {
	UserID int64
}

type RequestGetSubscription struct {
	UserID int64
}
