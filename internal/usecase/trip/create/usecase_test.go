package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/usecase/trip/create"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validReq := create.Request{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   trip.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1450,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		repo.EXPECT().InsertHistory(ctx, gomock.Any()).Return(nil)

		uc := create.New(repo, tx)
		res, err := uc.Execute(ctx, validReq)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, validReq.Price, res.Price)
	})

	t.Run("driver busy", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().Save(ctx, gomock.Any()).Return(trip.ErrDriverBusy)

		uc := create.New(repo, tx)
		res, err := uc.Execute(ctx, validReq)
		require.ErrorIs(t, err, trip.ErrDriverBusy)
		require.Nil(t, res)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		badReq := validReq
		badReq.Price = -10

		uc := create.New(repo, tx)
		res, err := uc.Execute(ctx, badReq)
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("db internal error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		tx := NewMocktxManager(ctrl)

		dbErr := errors.New("connection lost")
		tx.EXPECT().Do(ctx, gomock.Any()).Return(dbErr)

		uc := create.New(repo, tx)
		res, err := uc.Execute(ctx, validReq)
		require.ErrorIs(t, err, dbErr)
		require.Nil(t, res)
	})
}
