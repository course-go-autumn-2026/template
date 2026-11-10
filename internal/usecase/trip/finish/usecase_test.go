package finish_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"

	tripdomain "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/usecase/trip/finish"
)

func TestFinishSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)

	uc := finish.New(repo, tx, history, nil)

	id := uuid.New()
	finished := tripdomain.Trip{Id: id, Status: tripdomain.StatusCompleted}
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(finished, nil)
	history.EXPECT().Append(gomock.Any(), id, gomock.Any(), tripdomain.StatusCompleted, gomock.Any()).Return(nil)

	got, err := uc.Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != tripdomain.StatusCompleted {
		t.Fatalf("status = %v", got.Status)
	}
}

func TestFinishAlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)

	uc := finish.New(repo, tx, history, nil)

	id := uuid.New()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(tripdomain.Trip{}, tripdomain.ErrTripCompleted)

	_, err := uc.Execute(context.Background(), id)
	if !errors.Is(err, tripdomain.ErrTripCompleted) {
		t.Fatalf("expected ErrTripCompleted, got %v", err)
	}
}

func TestFinishNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)

	uc := finish.New(repo, tx, history, nil)

	id := uuid.New()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).Return(tripdomain.Trip{}, tripdomain.ErrTripNotFound)

	_, err := uc.Execute(context.Background(), id)
	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("expected ErrTripNotFound, got %v", err)
	}
}
