package dto

import "time"

type ResponseActivateSubscription struct {
	UserID int64
	EndAt  time.Time
}

type ResponseCancelSubscription struct {
	UserID int64
}

type ResponseGetSubscription struct {
	UserID  int64
	Active  bool
	StartAt time.Time
	EndAt   time.Time
}
