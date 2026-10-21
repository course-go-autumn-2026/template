package trip

import (
	"errors"
	"testing"
	"time"
)

func TestFinish(t *testing.T) {
	at := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, status string
		wantErr      error
	}{
		{"active", StatusActive, nil},
		{"completed", StatusCompleted, ErrTripCompleted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trip := Trip{Status: tc.status}
			err := trip.Finish(at)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Finish() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (trip.Status != StatusCompleted || trip.FinishedAt == nil || !trip.FinishedAt.Equal(at)) {
				t.Fatalf("Finish() did not complete trip: %+v", trip)
			}
		})
	}
}

func TestCoordinatesValidate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		point Coordinates
		valid bool
	}{
		{"valid", Coordinates{90, -180}, true},
		{"latitude", Coordinates{91, 0}, false},
		{"longitude", Coordinates{0, 181}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.point.Validate("point")
			if (err == nil) != tc.valid {
				t.Fatalf("Validate() = %v, valid = %v", err, tc.valid)
			}
		})
	}
}
