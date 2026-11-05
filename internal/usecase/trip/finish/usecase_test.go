package finish_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		activeTrip := &model.Trip{ID: id, Status: model.StatusActive}
		completedTrip := &model.Trip{ID: id, Status: model.StatusCompleted}

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().GetByID(ctx, id).Return(activeTrip, nil)
		repo.EXPECT().Finish(ctx, id, gomock.Any()).Return(completedTrip, nil)
		repo.EXPECT().InsertHistory(ctx, gomock.Any()).Return(nil)

		uc := finish.New(repo, tx)
		res, err := uc.Execute(ctx, id)
		require.NoError(t, err)
		require.Equal(t, model.StatusCompleted, res.Status)
	})

	t.Run("already completed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		completedTrip := &model.Trip{ID: id, Status: model.StatusCompleted}

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().GetByID(ctx, id).Return(completedTrip, nil)

		uc := finish.New(repo, tx)
		res, err := uc.Execute(ctx, id)
		require.ErrorIs(t, err, trip.ErrTripCompleted)
		require.Nil(t, res)
	})
}
