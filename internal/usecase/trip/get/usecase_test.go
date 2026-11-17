package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/usecase/trip/get"
)

func TestExecute(t *testing.T) {
	for _, failure := range []error{nil, trip.ErrTripNotFound} {
		t.Run(caseName(failure), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			ctx := context.Background()
			id := uuid.New()
			entity := trip.Trip{ID: id, Status: trip.StatusActive}
			repo.EXPECT().GetTrip(ctx, id).Return(entity, failure)

			result, err := get.New(repo).Execute(ctx, id)
			if !errors.Is(err, failure) {
				t.Fatalf("unexpected error: %v", err)
			}

			if failure == nil && result != entity {
				t.Fatalf("unexpected result: %+v", result)
			}

			if failure != nil && result.ID != uuid.Nil {
				t.Fatal("error returned a trip")
			}
		})
	}
}

func caseName(err error) string {
	if err == nil {
		return "success"
	}

	return err.Error()
}
