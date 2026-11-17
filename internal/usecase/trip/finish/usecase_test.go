package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/usecase/trip/finish"
)

type transactionKey struct{}

func TestExecute(t *testing.T) {
	cases := []struct {
		name   string
		status trip.Status
		want   error
	}{
		{"active trip", trip.StatusActive, nil},
		{"completed trip", trip.StatusCompleted, trip.ErrTripCompleted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			clock := NewMockclock(ctrl)
			observer := NewMockobserver(ctrl)

			ctx := context.Background()
			txCtx := context.WithValue(ctx, transactionKey{}, "transaction")
			at := time.Date(2026, 10, 27, 1, 2, 3, 0, time.UTC)
			id := uuid.New()
			entity := trip.Trip{ID: id, Status: tc.status}

			if tc.status == trip.StatusCompleted {
				entity.FinishedAt = &at
			}

			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				return fn(txCtx)
			})

			repo.EXPECT().GetTripForUpdate(txCtx, id).Return(entity, nil)
			clock.EXPECT().Now().Return(at)

			if tc.want == nil {
				previous := trip.StatusActive
				repo.EXPECT().FinishTrip(txCtx, id, at).Return(nil)
				repo.EXPECT().AddStatusHistory(txCtx, id, &previous, trip.StatusCompleted, "trip finished").Return(nil)
				observer.EXPECT().TripCompleted(ctx)
			}

			usecase := finish.New(repo, tx, clock, observer, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			actual, err := usecase.Execute(ctx, id)
			if !errors.Is(err, tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.want == nil && (actual.ID != id || actual.Status != trip.StatusCompleted || actual.FinishedAt == nil || !actual.FinishedAt.Equal(at)) {
				t.Fatalf("unexpected completed trip: %+v", actual)
			}

			if tc.want != nil && actual.ID != uuid.Nil {
				t.Fatal("failed operation returned a trip")
			}
		})
	}
}
