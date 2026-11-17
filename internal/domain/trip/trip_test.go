package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

func validData() trip.Data {
	return trip.Data{UserID: uuid.New(), DriverID: uuid.New(), StartPoint: trip.Coordinates{Latitude: 90, Longitude: -180}, EndPoint: trip.Coordinates{Latitude: -90, Longitude: 180}}
}

func TestLifecycle(t *testing.T) {
	id := uuid.New()
	data := validData()
	at := time.Date(2026, 10, 27, 12, 0, 0, 0, time.FixedZone("local", 18000))

	entity, err := trip.New(id, data, at)
	if err != nil {
		t.Fatal(err)
	}

	if entity.ID != id || entity.Data != data || entity.Status != trip.StatusActive || entity.FinishedAt != nil || !entity.StartedAt.Equal(at) || entity.StartedAt.Location() != time.UTC {
		t.Fatalf("invalid new trip: %+v", entity)
	}

	end := at.Add(time.Minute)

	if err := entity.Finish(end); err != nil {
		t.Fatal(err)
	}

	if entity.Status != trip.StatusCompleted || entity.FinishedAt == nil || !entity.FinishedAt.Equal(end) || entity.FinishedAt.Location() != time.UTC {
		t.Fatalf("invalid completed trip: %+v", entity)
	}

	if err := entity.Finish(end.Add(time.Minute)); !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("unexpected repeated finish: %v", err)
	}

	if !entity.FinishedAt.Equal(end) {
		t.Fatal("rejected finish changed the timestamp")
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name, field string
		change      func(*trip.Data)
	}{
		{"user", "user_id", func(d *trip.Data) { d.UserID = uuid.Nil }},
		{"driver", "driver_id", func(d *trip.Data) { d.DriverID = uuid.Nil }},
		{"price", "price", func(d *trip.Data) { d.Price = -1 }},
		{"start latitude", "start_point.latitude", func(d *trip.Data) { d.StartPoint.Latitude = 91 }},
		{"start longitude", "start_point.longitude", func(d *trip.Data) { d.StartPoint.Longitude = -181 }},
		{"end latitude", "end_point.latitude", func(d *trip.Data) { d.EndPoint.Latitude = -91 }},
		{"end longitude", "end_point.longitude", func(d *trip.Data) { d.EndPoint.Longitude = 181 }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := validData()
			tc.change(&data)

			entity, err := trip.New(uuid.New(), data, time.Now())

			var validation *trip.ValidationError

			if !errors.As(err, &validation) || validation.Field != tc.field || validation.Error() == "" {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if entity.ID != uuid.Nil {
				t.Fatal("invalid creation returned an entity")
			}
		})
	}
}
