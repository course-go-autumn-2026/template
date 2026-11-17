package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/freyliee/template/internal/domain/trip"
)

func TestPositionValidation(t *testing.T) {
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name     string
		point    trip.Coordinates
		recorded time.Time
		valid    bool
	}{
		{"boundaries", trip.Coordinates{Latitude: -90, Longitude: 180}, now, true},
		{"minute ahead", trip.Coordinates{}, now.Add(time.Minute), true},
		{"late arrival", trip.Coordinates{}, now.Add(-time.Hour), true},
		{"latitude", trip.Coordinates{Latitude: 91}, now, false},
		{"longitude", trip.Coordinates{Longitude: -181}, now, false},
		{"nan", trip.Coordinates{Latitude: math.NaN()}, now, false},
		{"infinity", trip.Coordinates{Longitude: math.Inf(1)}, now, false},
		{"future", trip.Coordinates{}, now.Add(time.Minute + time.Nanosecond), false},
		{"missing time", trip.Coordinates{}, time.Time{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (trip.PositionData{Coordinates: tc.point, RecordedAt: tc.recorded}).Validate(now)

			var validation *trip.ValidationError

			if tc.valid && err != nil {
				t.Fatal(err)
			}

			if !tc.valid && !errors.As(err, &validation) {
				t.Fatalf("expected validation error: %v", err)
			}
		})
	}
}
