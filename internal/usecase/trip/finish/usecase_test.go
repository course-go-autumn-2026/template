package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
	"github.com/eternal-despair/template/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	tripID := uuid.New()
	finishTime := time.Now().UTC()
	sampleFinishedTrip := &model.Trip{
		ID:         tripID,
		Status:     trip.StatusCompleted,
		FinishedAt: &finishTime,
	}

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success finish active trip", func(t *testing.T) {
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
			Finish(gomock.Any(), tripID, gomock.Any()).
			Return(sampleFinishedTrip, nil)

		repoMock.EXPECT().
			CreateStatusHistory(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, h *model.TripStatusHistory) error {
				assert.Equal(t, tripID, h.TripID)
				assert.Equal(t, string(trip.StatusCompleted), h.ToStatus)
				return nil
			})

		metricsMock.EXPECT().TripCompleted(gomock.Any())

		uc := finish.New(repoMock, txMgrMock, discardLogger, metricsMock)
		res, err := uc.Execute(context.Background(), tripID)

		require.NoError(t, err)
		assert.Equal(t, sampleFinishedTrip, res)
	})

	t.Run("trip already completed returns ErrTripCompleted", func(t *testing.T) {
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
			Finish(gomock.Any(), tripID, gomock.Any()).
			Return(nil, trip.ErrTripCompleted)

		uc := finish.New(repoMock, txMgrMock, discardLogger, nil)
		res, err := uc.Execute(context.Background(), tripID)

		require.ErrorIs(t, err, trip.ErrTripCompleted)
		assert.Nil(t, res)
	})

	t.Run("trip not found returns ErrTripNotFound", func(t *testing.T) {
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
			Finish(gomock.Any(), tripID, gomock.Any()).
			Return(nil, trip.ErrTripNotFound)

		uc := finish.New(repoMock, txMgrMock, discardLogger, nil)
		res, err := uc.Execute(context.Background(), tripID)

		require.ErrorIs(t, err, trip.ErrTripNotFound)
		assert.Nil(t, res)
	})

	t.Run("empty id returns validation error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)

		uc := finish.New(repoMock, txMgrMock, discardLogger, nil)
		res, err := uc.Execute(context.Background(), uuid.Nil)

		require.Error(t, err)
		var valErr *trip.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Equal(t, "id", valErr.Field)
		assert.Nil(t, res)
	})

	t.Run("repository error aborts transaction", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)
		txMgrMock := NewMocktxManager(ctrl)

		dbErr := errors.New("db error")
		txMgrMock.EXPECT().
			Do(gomock.Any(), gomock.Any()).
			Return(dbErr)

		uc := finish.New(repoMock, txMgrMock, discardLogger, nil)
		res, err := uc.Execute(context.Background(), tripID)

		require.ErrorIs(t, err, dbErr)
		assert.Nil(t, res)
	})
}
