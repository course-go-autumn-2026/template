package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/XrystikLK/template/internal/domain/trip"
)

func TestNewTrip(t *testing.T) {
	now := time.Now().UTC()
	validUserID := uuid.New()
	validDriverID := uuid.New()
	validStart := trip.Coordinates{Latitude: 55.7558, Longitude: 37.6173}
	validEnd := trip.Coordinates{Latitude: 59.9343, Longitude: 30.3351}

	tests := []struct {
		name     string
		userID   uuid.UUID
		driverID uuid.UUID
		start    trip.Coordinates
		end      trip.Coordinates
		price    int64
		wantErr  bool
		errField string
	}{
		{
			name:     "valid trip",
			userID:   validUserID,
			driverID: validDriverID,
			start:    validStart,
			end:      validEnd,
			price:    500,
			wantErr:  false,
		},
		{
			name:     "empty user id",
			userID:   uuid.Nil,
			driverID: validDriverID,
			start:    validStart,
			end:      validEnd,
			price:    500,
			wantErr:  true,
			errField: "user_id",
		},
		{
			name:     "empty driver id",
			userID:   validUserID,
			driverID: uuid.Nil,
			start:    validStart,
			end:      validEnd,
			price:    500,
			wantErr:  true,
			errField: "driver_id",
		},
		{
			name:     "negative price",
			userID:   validUserID,
			driverID: validDriverID,
			start:    validStart,
			end:      validEnd,
			price:    -100,
			wantErr:  true,
			errField: "price",
		},
		{
			name:     "invalid start latitude",
			userID:   validUserID,
			driverID: validDriverID,
			start:    trip.Coordinates{Latitude: 95.0, Longitude: 37.6173},
			end:      validEnd,
			price:    500,
			wantErr:  true,
			errField: "start_point",
		},
		{
			name:     "invalid end longitude",
			userID:   validUserID,
			driverID: validDriverID,
			start:    validStart,
			end:      trip.Coordinates{Latitude: 59.9343, Longitude: 185.0},
			price:    500,
			wantErr:  true,
			errField: "end_point",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := trip.New(tt.userID, tt.driverID, tt.start, tt.end, tt.price, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var valErr *trip.ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				if valErr.Field != tt.errField {
					t.Fatalf("expected field %s, got %s", tt.errField, valErr.Field)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatalf("expected trip, got nil")
			}
			if got.Status != trip.StatusActive {
				t.Fatalf("expected status %s, got %s", trip.StatusActive, got.Status)
			}
			if got.Price != tt.price {
				t.Fatalf("expected price %d, got %d", tt.price, got.Price)
			}
		})
	}
}

func TestTripFinish(t *testing.T) {
	now := time.Now().UTC()
	finishTime := now.Add(20 * time.Minute)

	t.Run("finish active trip successfully", func(t *testing.T) {
		tr, err := trip.New(
			uuid.New(),
			uuid.New(),
			trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
			trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
			350,
			now,
		)
		if err != nil {
			t.Fatalf("failed to create trip: %v", err)
		}

		if err := tr.Finish(finishTime); err != nil {
			t.Fatalf("unexpected error on finish: %v", err)
		}

		if tr.Status != trip.StatusCompleted {
			t.Fatalf("expected status %s, got %s", trip.StatusCompleted, tr.Status)
		}
		if tr.FinishedAt == nil || !tr.FinishedAt.Equal(finishTime) {
			t.Fatalf("expected finishedAt %v, got %v", finishTime, tr.FinishedAt)
		}
	})

	t.Run("finish already completed trip returns ErrTripCompleted", func(t *testing.T) {
		tr, err := trip.New(
			uuid.New(),
			uuid.New(),
			trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
			trip.Coordinates{Latitude: 55.80, Longitude: 37.70},
			350,
			now,
		)
		if err != nil {
			t.Fatalf("failed to create trip: %v", err)
		}

		if err := tr.Finish(finishTime); err != nil {
			t.Fatalf("unexpected error on first finish: %v", err)
		}

		err = tr.Finish(finishTime.Add(5 * time.Minute))
		if !errors.Is(err, trip.ErrTripCompleted) {
			t.Fatalf("expected ErrTripCompleted, got: %v", err)
		}
	})
}

func TestRestoreTrip(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()
	start := trip.Coordinates{Latitude: 55.75, Longitude: 37.61}
	end := trip.Coordinates{Latitude: 55.80, Longitude: 37.70}
	now := time.Now().UTC()
	finishedAt := now.Add(10 * time.Minute)

	tr := trip.Restore(
		id, userID, driverID,
		start, end,
		500,
		trip.StatusCompleted,
		now,
		&finishedAt,
		now,
		finishedAt,
	)

	if tr == nil {
		t.Fatal("expected restored trip, got nil")
	}
	if tr.ID != id || tr.UserID != userID || tr.DriverID != driverID {
		t.Errorf("ids mismatch")
	}
	if tr.Status != trip.StatusCompleted {
		t.Errorf("expected status completed, got %s", tr.Status)
	}
	if tr.Price != 500 {
		t.Errorf("expected price 500, got %d", tr.Price)
	}
	if tr.FinishedAt == nil || !tr.FinishedAt.Equal(finishedAt) {
		t.Errorf("finishedAt mismatch")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &trip.ValidationError{
		Field:   "price",
		Message: "must be positive",
	}
	expected := "validation error for field price: must be positive"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}
