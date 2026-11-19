package create_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/usecase/position/create"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tripID := uuid.New()
	now := time.Now().UTC()

	validReq := create.Request{
		TripID:      tripID,
		Coordinates: model.Point{Latitude: 59.935, Longitude: 30.335},
		RecordedAt:  now,
	}

	t.Run("success cancels queued poll", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMockpositionRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		canceller := NewMockpollCanceller(ctrl)

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().UpdateLastPositionAt(ctx, tripID, now).Return(nil)
		repo.EXPECT().InsertPosition(ctx, gomock.Any()).Return(int64(42), nil)
		canceller.EXPECT().Cancel(tripID)

		uc := create.New(repo, tx, canceller)
		res, err := uc.Execute(ctx, validReq)
		require.NoError(t, err)
		require.Equal(t, int64(42), res.ID)
		require.Equal(t, tripID, res.TripID)
	})

	t.Run("trip completed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMockpositionRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		canceller := NewMockpollCanceller(ctrl)

		tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(c context.Context, fn func(context.Context) error) error {
			return fn(c)
		})
		repo.EXPECT().UpdateLastPositionAt(ctx, tripID, now).Return(domaintrip.ErrTripCompleted)

		uc := create.New(repo, tx, canceller)
		res, err := uc.Execute(ctx, validReq)
		require.ErrorIs(t, err, domaintrip.ErrTripCompleted)
		require.Nil(t, res)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := NewMockpositionRepository(ctrl)
		tx := NewMocktxManager(ctrl)
		canceller := NewMockpollCanceller(ctrl)

		badReq := validReq
		badReq.Coordinates.Latitude = 120

		uc := create.New(repo, tx, canceller)
		res, err := uc.Execute(ctx, badReq)
		var valErr *domaintrip.ValidationError
		require.ErrorAs(t, err, &valErr)
		require.Nil(t, res)
	})
}
