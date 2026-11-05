package create_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	"github.com/ggwpfu9/courseAvito/internal/usecase/trip/create"
)

func TestCreateUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockTripRepository(ctrl)
	mockTx := NewMockTxManager(ctrl)

	uc := create.New(mockRepo, mockTx)

	trip := appmodel.Trip{
		UserID:   "user-1",
		DriverID: "driver-1",
		Price:    1450,
	}

	t.Run("success", func(t *testing.T) {
		mockTx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		)

		mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		result, err := uc.Execute(context.Background(), trip)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID == "" {
			t.Errorf("expected trip ID to be set")
		}
	})

	t.Run("error from repo", func(t *testing.T) {
		mockTx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		)

		mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

		_, err := uc.Execute(context.Background(), trip)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}