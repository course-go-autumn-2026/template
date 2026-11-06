package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/domain/trip"
)

func TestTrip_Finish(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		status  trip.Status
		wantErr error
		check   func(t *testing.T, tr *trip.Trip)
	}{
		{
			name:    "active trip finishes successfully",
			status:  trip.StatusActive,
			wantErr: nil,
			check: func(t *testing.T, tr *trip.Trip) {
				if tr.Status != trip.StatusCompleted {
					t.Fatalf("status = %q, want completed", tr.Status)
				}
				if tr.FinishedAt == nil || !tr.FinishedAt.Equal(now) {
					t.Fatalf("finishedAt not set")
				}
			},
		},
		{
			name:    "already completed returns ErrTripCompleted",
			status:  trip.StatusCompleted,
			wantErr: trip.ErrTripCompleted,
			check: func(t *testing.T, tr *trip.Trip) {
				if tr.Status != trip.StatusCompleted {
					t.Fatalf("status changed on error")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &trip.Trip{Status: tt.status}
			err := tr.Finish(now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			tt.check(t, tr)
		})
	}
}

func TestTrip_Validate(t *testing.T) {
	valid := trip.Trip{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Start:     trip.Point{Latitude: 59.9, Longitude: 30.3},
		End:       trip.Point{Latitude: 59.8, Longitude: 30.4},
		Price:     100,
		Status:    trip.StatusActive,
		StartedAt: time.Now(),
	}

	tests := []struct {
		name    string
		mutate  func(*trip.Trip)
		wantErr bool
		field   string
	}{
		{name: "valid", mutate: func(*trip.Trip) {}},
		{name: "nil id", mutate: func(tr *trip.Trip) { tr.ID = uuid.Nil }, wantErr: true, field: "id"},
		{name: "negative price", mutate: func(tr *trip.Trip) { tr.Price = -1 }, wantErr: true, field: "price"},
		{name: "lat out of range", mutate: func(tr *trip.Trip) { tr.Start.Latitude = 100 }, wantErr: true, field: "start_point.latitude"},
		{name: "lon out of range", mutate: func(tr *trip.Trip) { tr.End.Longitude = -200 }, wantErr: true, field: "end_point.longitude"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := valid
			tt.mutate(&tr)
			err := tr.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				var ve *trip.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if ve.Field != tt.field {
					t.Fatalf("field = %q, want %q", ve.Field, tt.field)
				}
			}
		})
	}
}

func TestTrip_Validate_Boundaries(t *testing.T) {
	valid := trip.Trip{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Start:     trip.Point{Latitude: 0, Longitude: 0},
		End:       trip.Point{Latitude: 0, Longitude: 0},
		Price:     0,
		Status:    trip.StatusActive,
		StartedAt: time.Now(),
	}

	tests := []struct {
		name    string
		mutate  func(*trip.Trip)
		wantErr bool
		field   string
	}{
		{name: "price zero", mutate: func(tr *trip.Trip) { tr.Price = 0 }, wantErr: false},
		{name: "price negative one", mutate: func(tr *trip.Trip) { tr.Price = -1 }, wantErr: true, field: "price"},

		{name: "unknown status", mutate: func(tr *trip.Trip) { tr.Status = "invalid_status" }, wantErr: true, field: "status"},

		{name: "lat exactly -90", mutate: func(tr *trip.Trip) { tr.Start.Latitude = -90 }, wantErr: false},
		{name: "lat exactly 90", mutate: func(tr *trip.Trip) { tr.Start.Latitude = 90 }, wantErr: false},
		{name: "lat just above 90", mutate: func(tr *trip.Trip) { tr.Start.Latitude = 90.0001 }, wantErr: true, field: "start_point.latitude"},
		{name: "lon exactly -180", mutate: func(tr *trip.Trip) { tr.End.Longitude = -180 }, wantErr: false},
		{name: "lon exactly 180", mutate: func(tr *trip.Trip) { tr.End.Longitude = 180 }, wantErr: false},
		{name: "lon just below -180", mutate: func(tr *trip.Trip) { tr.End.Longitude = -180.0001 }, wantErr: true, field: "end_point.longitude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := valid
			tt.mutate(&tr)
			err := tr.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				var ve *trip.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if ve.Field != tt.field {
					t.Fatalf("field = %q, want %q", ve.Field, tt.field)
				}
			}
		})
	}
}
