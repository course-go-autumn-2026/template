package model
import "time"
type Trip struct {
    ID          string
    UserID      string
    DriverID    string
    StartLat     float64
    StartLng    float64
    EndLat      float64
    EndLng     float64
    Price       int64
    Status      string
    StartedAt   time.Time
    FinishedAt  *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}