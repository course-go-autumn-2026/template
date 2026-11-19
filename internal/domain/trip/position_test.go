package trip_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
)

func TestValidatePosition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		lat        float64
		lon        float64
		recordedAt time.Time
		wantErr    bool
	}{
		{
			name:       "valid current position",
			lat:        59.935,
			lon:        30.335,
			recordedAt: now,
			wantErr:    false,
		},
		{
			name:       "valid within 1 minute future skew",
			lat:        59.935,
			lon:        30.335,
			recordedAt: now.Add(30 * time.Second),
			wantErr:    false,
		},
		{
			name:       "invalid latitude out of range",
			lat:        91,
			lon:        30.335,
			recordedAt: now,
			wantErr:    true,
		},
		{
			name:       "invalid longitude NaN",
			lat:        59.935,
			lon:        math.NaN(),
			recordedAt: now,
			wantErr:    true,
		},
		{
			name:       "zero recorded_at",
			lat:        59.935,
			lon:        30.335,
			recordedAt: time.Time{},
			wantErr:    true,
		},
		{
			name:       "recorded_at more than 1 minute in future",
			lat:        59.935,
			lon:        30.335,
			recordedAt: now.Add(61 * time.Second),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := trip.ValidatePosition(tt.lat, tt.lon, tt.recordedAt, now)
			if tt.wantErr {
				var valErr *trip.ValidationError
				require.ErrorAs(t, err, &valErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
