package list_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/usecase/position/list"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tripID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMockpositionRepository(ctrl)

		expected := []model.Position{
			{ID: 1, TripID: tripID, RecordedAt: time.Now().UTC()},
		}
		repo.EXPECT().ListPositions(ctx, tripID).Return(expected, nil)

		uc := list.New(repo)
		res, err := uc.Execute(ctx, tripID)
		require.NoError(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMockpositionRepository(ctrl)

		repo.EXPECT().ListPositions(ctx, tripID).Return(nil, domaintrip.ErrTripNotFound)

		uc := list.New(repo)
		res, err := uc.Execute(ctx, tripID)
		require.ErrorIs(t, err, domaintrip.ErrTripNotFound)
		require.Nil(t, res)
	})
}
