package get_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	"github.com/ggwpfu9/courseAvito/internal/usecase/trip/get"
)

func TestGetUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockTripRepository(ctrl)
	uc := get.New(mockRepo)

	tripID := "some-uuid"

	t.Run("success", func(t *testing.T) {
		expectedTrip := &appmodel.Trip{ID: tripID, Status: "active"}
		mockRepo.EXPECT().GetByID(gomock.Any(), tripID).Return(expectedTrip, nil)

		result, err := uc.Execute(context.Background(), tripID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != tripID {
			t.Errorf("expected ID %s, got %s", tripID, result.ID)
		}
	})

	t.Run("error from repo", func(t *testing.T) {
		mockRepo.EXPECT().GetByID(gomock.Any(), tripID).Return(nil, errors.New("db error"))

		_, err := uc.Execute(context.Background(), tripID)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}