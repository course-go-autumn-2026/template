package model

import "time"

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         string
	UserID     string
	DriverID   string
	StartPoint Point
	EndPoint   Point
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}