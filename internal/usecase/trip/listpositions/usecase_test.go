package listpositions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/usecase/trip/listpositions"
)

func TestExecute(t *testing.T) {
	failure := errors.New("storage failed")

	for _, tc := range []struct {
		name            string
		getErr, listErr error
	}{
		{name: "success"}, {name: "missing", getErr: trip.ErrTripNotFound}, {name: "storage error", listErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMocktripRepository(gomock.NewController(t))
			ctx, id := context.Background(), uuid.New()
			repo.EXPECT().GetTrip(ctx, id).Return(trip.Trip{}, tc.getErr)
			wantErr := tc.getErr
			expected := []trip.Position{{ID: 42, TripID: id}}

			if tc.getErr == nil {
				repo.EXPECT().ListPositions(ctx, id).Return(expected, tc.listErr)
				wantErr = tc.listErr
			}

			actual, err := listpositions.New(repo).Execute(ctx, id)

			if !errors.Is(err, wantErr) {
				t.Fatal(err)
			}

			if err == nil && (len(actual) != 1 || actual[0] != expected[0]) {
				t.Fatalf("route: %+v", actual)
			}
		})
	}
}
