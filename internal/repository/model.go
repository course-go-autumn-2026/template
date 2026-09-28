package repository

import (
	"time"
)

type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Trip struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	DriverID       string     `json:"driver_id"`
	StartPoint     Point      `json:"start_point"`
	EndPoint       Point      `json:"end_point"`
	Price          int64      `json:"price"` // В целых рублях
	Status         string     `json:"status"` // active, completed
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}
