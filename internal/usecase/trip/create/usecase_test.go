package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
	"github.com/eternal-despair/template/internal/usecase/trip/create"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	validUser := uuid.New()
	validDriver := uuid.New()
	validInput := model.CreateTripInput{
		UserID:     validUser,
		DriverID:   validDriver,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      500,
	}

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success create trip", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)
		metricsMock := NewMockmetrics(ctrl)

		txMgrMock.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repoMock.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, tr *model.Trip) error {
				assert.Equal(t, validUser, tr.UserID)
				assert.Equal(t, validDriver, tr.DriverID)
				assert.Equal(t, trip.StatusActive, tr.Status)
				return nil
			})

		repoMock.EXPECT().
			CreateStatusHistory(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, h *model.TripStatusHistory) error {
				assert.Equal(t, string(trip.StatusActive), h.ToStatus)
				return nil
			})

		metricsMock.EXPECT().TripCreated(gomock.Any())

		uc := create.New(repoMock, txMgrMock, discardLogger, metricsMock)
		created, err := uc.Execute(context.Background(), validInput)

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, validUser, created.UserID)
		assert.Equal(t, validDriver, created.DriverID)
	})

	t.Run("driver busy returns ErrDriverBusy", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)

		txMgrMock.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			})

		repoMock.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(trip.ErrDriverBusy)

		uc := create.New(repoMock, txMgrMock, discardLogger, nil)
		created, err := uc.Execute(context.Background(), validInput)

		require.ErrorIs(t, err, trip.ErrDriverBusy)
		assert.Nil(t, created)
	})

	t.Run("validation error does not call txManager or repository", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)

		uc := create.New(repoMock, txMgrMock, discardLogger, nil)
		invalidInput := validInput
		invalidInput.UserID = uuid.Nil

		created, err := uc.Execute(context.Background(), invalidInput)

		require.Error(t, err)
		var valErr *trip.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Equal(t, "user_id", valErr.Field)
		assert.Nil(t, created)
	})

	t.Run("repository error aborts transaction", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)

		dbErr := errors.New("db connection failure")
		txMgrMock.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			Return(dbErr)

		uc := create.New(repoMock, txMgrMock, discardLogger, nil)
		created, err := uc.Execute(context.Background(), validInput)

		require.ErrorIs(t, err, dbErr)
		assert.Nil(t, created)
	})
}
