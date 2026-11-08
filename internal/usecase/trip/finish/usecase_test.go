package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/model"
	"github.com/XrystikLK/template/internal/usecase/trip/finish"
)

func TestFinishTripUseCase(t *testing.T) {
	tripID := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()
	now := time.Now().UTC()

	activeTrip := &model.Trip{
		ID:         tripID,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   model.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      500,
		Status:     model.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		logger := NewMocklogger(ctrl)
		metrics := NewMockmetrics(ctrl)

		expectedFinished := &model.Trip{
			ID:         tripID,
			UserID:     userID,
			DriverID:   driverID,
			StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.61},
			EndPoint:   model.Coordinates{Latitude: 55.80, Longitude: 37.70},
			Price:      500,
			Status:     model.StatusCompleted,
			StartedAt:  now,
			FinishedAt: &now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(activeTrip, nil)

		repo.EXPECT().
			Finish(gomock.Any(), tripID, gomock.Any()).
			Return(expectedFinished, nil)

		repo.EXPECT().
			AddStatusHistory(gomock.Any(), gomock.AssignableToTypeOf(model.StatusHistory{})).
			Return(nil)

		logger.EXPECT().
			InfoContext(gomock.Any(), "trip completed", gomock.Any()).
			Times(1)

		metrics.EXPECT().
			IncTripsCompleted(gomock.Any()).
			Times(1)

		uc := finish.New(repo, tx, logger, metrics)
		res, err := uc.Execute(context.Background(), tripID)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if res == nil || res.Status != model.StatusCompleted {
			t.Fatalf("expected completed status, got: %v", res)
		}
	})

	t.Run("trip already completed in domain check", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		completedTrip := *activeTrip
		completedTrip.Status = model.StatusCompleted
		completedTrip.FinishedAt = &now

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(&completedTrip, nil)

		uc := finish.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), tripID)

		if !errors.Is(err, domainTrip.ErrTripCompleted) {
			t.Fatalf("expected ErrTripCompleted, got: %v", err)
		}
	})

	t.Run("trip not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(nil, domainTrip.ErrTripNotFound)

		uc := finish.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), tripID)

		if !errors.Is(err, domainTrip.ErrTripNotFound) {
			t.Fatalf("expected ErrTripNotFound, got: %v", err)
		}
	})

	t.Run("repository unexpected error on finish", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		dbErr := errors.New("write timeout")

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(activeTrip, nil)

		repo.EXPECT().
			Finish(gomock.Any(), tripID, gomock.Any()).
			Return(nil, dbErr)

		uc := finish.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), tripID)

		if !errors.Is(err, dbErr) {
			t.Fatalf("expected dbErr, got: %v", err)
		}
	})
}
