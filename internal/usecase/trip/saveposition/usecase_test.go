package saveposition_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/usecase/trip/saveposition"
)

func TestExecute(t *testing.T) {
	failure := errors.New("storage failed")

	for _, tc := range []struct {
		name                        string
		status                      trip.Status
		readErr, saveErr, commitErr error
	}{
		{name: "success", status: trip.StatusActive},
		{name: "missing", readErr: trip.ErrTripNotFound},
		{name: "completed", status: trip.StatusCompleted},
		{name: "save failure", status: trip.StatusActive, saveErr: failure},
		{name: "commit failure", status: trip.StatusActive, commitErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo, tx, clock, canceller, observer := NewMocktripRepository(ctrl), NewMocktxManager(ctrl), NewMockclock(ctrl), NewMockcanceller(ctrl), NewMockobserver(ctrl)
			ctx := context.Background()
			id := uuid.New()
			now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
			data := trip.PositionData{Coordinates: trip.Coordinates{Latitude: 12, Longitude: 34}, RecordedAt: now}
			expected := trip.Position{ID: 42, TripID: id, PositionData: data}
			clock.EXPECT().Now().Return(now)
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				if err := fn(ctx); err != nil {
					return err
				}

				return tc.commitErr
			})
			repo.EXPECT().GetTripForUpdate(ctx, id).Return(trip.Trip{Status: tc.status}, tc.readErr)
			wantErr := tc.readErr

			if wantErr == nil && tc.status == trip.StatusCompleted {
				wantErr = trip.ErrTripCompleted
			}

			if wantErr == nil {
				repo.EXPECT().SavePosition(ctx, id, data).Return(expected, tc.saveErr)
				wantErr = tc.saveErr

				if wantErr == nil {
					wantErr = tc.commitErr
				}
			}

			if wantErr == nil {
				gomock.InOrder(canceller.EXPECT().Cancel(id), observer.EXPECT().PositionSaved(ctx))
			}

			actual, err := saveposition.New(repo, tx, clock, canceller, observer).Execute(ctx, id, data)

			if !errors.Is(err, wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}

			if err == nil && actual != expected {
				t.Fatalf("position %+v", actual)
			}

			if err != nil && actual.ID != 0 {
				t.Fatal("failed operation returned position")
			}
		})
	}
}

func TestValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	clock := NewMockclock(ctrl)
	clock.EXPECT().Now().Return(time.Now())
	_, err := saveposition.New(nil, nil, clock, nil, nil).Execute(context.Background(), uuid.New(), trip.PositionData{})

	var validation *trip.ValidationError

	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error: %v", err)
	}
}
