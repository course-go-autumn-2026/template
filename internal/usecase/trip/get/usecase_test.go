package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
	"github.com/eternal-despair/template/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	tripID := uuid.New()
	sampleTrip := &model.Trip{
		ID:        tripID,
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Status:    trip.StatusActive,
		CreatedAt: time.Now().UTC(),
	}

	t.Run("success get trip", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)

		repoMock.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(sampleTrip, nil)

		uc := get.New(repoMock)
		found, err := uc.Execute(context.Background(), tripID)

		require.NoError(t, err)
		assert.Equal(t, sampleTrip, found)
	})

	t.Run("trip not found returns ErrTripNotFound", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)

		repoMock.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(nil, trip.ErrTripNotFound)

		uc := get.New(repoMock)
		found, err := uc.Execute(context.Background(), tripID)

		require.ErrorIs(t, err, trip.ErrTripNotFound)
		assert.Nil(t, found)
	})

	t.Run("empty id returns validation error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)

		uc := get.New(repoMock)
		found, err := uc.Execute(context.Background(), uuid.Nil)

		require.Error(t, err)
		var valErr *trip.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Equal(t, "id", valErr.Field)
		assert.Nil(t, found)
	})

	t.Run("repository error propagates", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repoMock := NewMocktripRepository(ctrl)

		dbErr := errors.New("db error")
		repoMock.EXPECT().
			GetByID(gomock.Any(), tripID).
			Return(nil, dbErr)

		uc := get.New(repoMock)
		found, err := uc.Execute(context.Background(), tripID)

		require.ErrorIs(t, err, dbErr)
		assert.Nil(t, found)
	})
}
