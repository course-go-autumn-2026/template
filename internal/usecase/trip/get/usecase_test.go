package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/model"
	"github.com/XrystikLK/template/internal/usecase/trip/get"
)

func TestGetTripUseCase(t *testing.T) {
	tripID := uuid.New()

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		expectedTrip := &model.Trip{
			ID:     tripID,
			Price:  1000,
			Status: model.StatusActive,
		}

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(expectedTrip, nil)

		uc := get.New(repo)
		res, err := uc.Execute(context.Background(), tripID)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if res == nil || res.ID != tripID {
			t.Fatalf("expected trip %v, got %v", expectedTrip, res)
		}
	})

	t.Run("trip not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(nil, domainTrip.ErrTripNotFound)

		uc := get.New(repo)
		_, err := uc.Execute(context.Background(), tripID)

		if !errors.Is(err, domainTrip.ErrTripNotFound) {
			t.Fatalf("expected ErrTripNotFound, got: %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		dbErr := errors.New("db error")

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(nil, dbErr)

		uc := get.New(repo)
		_, err := uc.Execute(context.Background(), tripID)

		if !errors.Is(err, dbErr) {
			t.Fatalf("expected dbErr, got: %v", err)
		}
	})
}
