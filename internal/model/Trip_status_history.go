package model

import "time"

type TripStatusHistory struct {
	ID         int64
	TripID     string
	FromStatus *string
	ToStatus   string
	Reason     *string
	ChangedAt  time.Time
}
