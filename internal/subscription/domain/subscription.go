package domain

import "time"

type Subscription struct {
	UserID  int64
	Active  bool
	StartAt time.Time
	EndAt   time.Time
}
