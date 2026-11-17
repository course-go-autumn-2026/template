package create_test

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
	"github.com/freyliee/template/internal/usecase/trip/create"
)

type transactionKey struct{}

func TestExecute(t *testing.T) {
	cases := []struct {
		name            string
		repositoryError error
	}{
		{"success", nil},
		{"driver busy", trip.ErrDriverBusy},
		{"repository error", errors.New("database unavailable")},
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
			input := trip.Data{UserID: uuid.New(), DriverID: uuid.New(), Price: 123}

			clock.EXPECT().Now().Return(at)
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
				return fn(txCtx)
			})

			var saved trip.Trip

			repo.EXPECT().CreateTrip(txCtx, gomock.Any()).DoAndReturn(func(_ context.Context, entity trip.Trip) error {
				saved = entity

				if entity.ID == uuid.Nil || entity.Data != input || entity.Status != trip.StatusActive || !entity.StartedAt.Equal(at) {
					t.Fatalf("invalid saved trip: %+v", entity)
				}

				return tc.repositoryError
			})

			if tc.repositoryError == nil {
				repo.EXPECT().AddStatusHistory(txCtx, gomock.Any(), nil, trip.StatusActive, "trip created").DoAndReturn(func(_ context.Context, id uuid.UUID, _ *trip.Status, _ trip.Status, _ string) error {
					if id != saved.ID {
						t.Fatal("history refers to another trip")
					}

					return nil
				})

				observer.EXPECT().TripCreated(ctx)
			}

			usecase := create.New(repo, tx, clock, observer, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			actual, err := usecase.Execute(ctx, input)
			if !errors.Is(err, tc.repositoryError) {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.repositoryError == nil && actual != saved {
				t.Fatalf("unexpected trip: %+v", actual)
			}

			if tc.repositoryError != nil && actual.ID != uuid.Nil {
				t.Fatal("failed operation returned a trip")
			}
		})
	}
}
