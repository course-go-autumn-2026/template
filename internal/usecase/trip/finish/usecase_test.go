package finish_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	tripDomain "github.com/ggwpfu9/courseAvito/internal/domain/trip"
	"github.com/ggwpfu9/courseAvito/internal/usecase/trip/finish"
)

func TestFinishUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockTripRepository(ctrl)
	mockTx := NewMockTxManager(ctrl)
	uc := finish.New(mockRepo, mockTx)

	tripID := "some-uuid"

	t.Run("success", func(t *testing.T) {
		activeTrip := &appmodel.Trip{ID: tripID, Status: "active"}

		mockTx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		)

		mockRepo.EXPECT().GetByID(gomock.Any(), tripID).Return(activeTrip, nil)
		mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Execute(context.Background(), tripID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "completed" {
			t.Errorf("expected status completed, got %s", result.Status)
		}
	})

	t.Run("error: already completed", func(t *testing.T) {
		completedTrip := &appmodel.Trip{ID: tripID, Status: "completed"}

		mockTx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		)

		mockRepo.EXPECT().GetByID(gomock.Any(), tripID).Return(completedTrip, nil)

		_, err := uc.Execute(context.Background(), tripID)
		if !errors.Is(err, tripDomain.ErrTripCompleted) {
			t.Errorf("expected ErrTripCompleted, got %v", err)
		}
	})
}