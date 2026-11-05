package trip_test

import (
	"errors"
	"testing"
	"time"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	tripDomain "github.com/ggwpfu9/courseAvito/internal/domain/trip"
)

func TestFinishTrip(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		trip      *appmodel.Trip
		wantErr   error
		wantStatus string
	}{
		{
			name: "success: active trip finishes",
			trip: &appmodel.Trip{
				Status: "active",
			},
			wantErr:   nil,
			wantStatus: "completed",
		},
		{
			name: "error: already completed trip",
			trip: &appmodel.Trip{
				Status: "completed",
			},
			wantErr:   tripDomain.ErrTripCompleted,
			wantStatus: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tripDomain.FinishTrip(tt.trip, now)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("FinishTrip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.trip.Status != tt.wantStatus {
				t.Errorf("FinishTrip() status = %v, wantStatus %v", tt.trip.Status, tt.wantStatus)
			}

			if tt.wantErr == nil && tt.trip.FinishedAt == nil {
				t.Errorf("FinishTrip() FinishedAt is nil, expected it to be set")
			}
		})
	}
}