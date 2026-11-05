package get_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		expected := &model.Trip{ID: id}
		repo.EXPECT().GetByID(ctx, id).Return(expected, nil)

		uc := get.New(repo)
		res, err := uc.Execute(ctx, id)
		require.NoError(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		repo.EXPECT().GetByID(ctx, id).Return(nil, trip.ErrTripNotFound)

		uc := get.New(repo)
		res, err := uc.Execute(ctx, id)
		require.ErrorIs(t, err, trip.ErrTripNotFound)
		require.Nil(t, res)
	})
}
