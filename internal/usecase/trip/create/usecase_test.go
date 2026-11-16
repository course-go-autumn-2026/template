package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/model"
	"github.com/XrystikLK/template/internal/usecase/trip/create"
)

func TestCreateTripUseCase(t *testing.T) {
	validUserID := uuid.New()
	validDriverID := uuid.New()
	validReq := create.Request{
		UserID:     validUserID,
		DriverID:   validDriverID,
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   model.Coordinates{Latitude: 55.80, Longitude: 37.70},
		Price:      500,
	}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		logger := NewMocklogger(ctrl)
		metrics := NewMockmetrics(ctrl)

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			Save(gomock.Any(), gomock.AssignableToTypeOf(model.Trip{})).
			Return(nil)

		repo.EXPECT().
			AddStatusHistory(gomock.Any(), gomock.AssignableToTypeOf(model.StatusHistory{})).
			Return(nil)

		logger.EXPECT().
			InfoContext(gomock.Any(), "trip created", gomock.Any()).
			Times(1)

		metrics.EXPECT().
			IncTripsCreated(gomock.Any()).
			Times(1)

		uc := create.New(repo, tx, logger, metrics)
		tripModel, err := uc.Execute(context.Background(), validReq)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if tripModel == nil {
			t.Fatalf("expected trip, got nil")
		}
		if tripModel.Status != model.StatusActive {
			t.Fatalf("expected active status, got: %v", tripModel.Status)
		}
	})

	t.Run("driver busy error", func(t *testing.T) {
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
			Save(gomock.Any(), gomock.Any()).
			Return(domainTrip.ErrDriverBusy)

		uc := create.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), validReq)

		if !errors.Is(err, domainTrip.ErrDriverBusy) {
			t.Fatalf("expected ErrDriverBusy, got: %v", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		invalidReq := validReq
		invalidReq.Price = -50

		uc := create.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), invalidReq)

		var valErr *domainTrip.ValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ValidationError, got: %v", err)
		}
	})

	t.Run("repository unexpected error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		dbErr := errors.New("connection failed")

		tx.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repo.EXPECT().
			Save(gomock.Any(), gomock.Any()).
			Return(dbErr)

		uc := create.New(repo, tx, nil, nil)
		_, err := uc.Execute(context.Background(), validReq)

		if !errors.Is(err, dbErr) {
			t.Fatalf("expected dbErr, got: %v", err)
		}
	})
}
