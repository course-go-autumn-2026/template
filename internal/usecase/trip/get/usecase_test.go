package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"

	tripdomain "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/usecase/trip/get"
)

func TestGetSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	uc := get.New(repo)

	id := uuid.New()
	stored := tripdomain.Trip{Id: id, Status: tripdomain.StatusActive}
	repo.EXPECT().Get(gomock.Any(), id).Return(stored, nil)

	got, err := uc.Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Id != id {
		t.Fatal("id mismatch")
	}
}

func TestGetNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	uc := get.New(repo)

	id := uuid.New()
	repo.EXPECT().Get(gomock.Any(), id).Return(tripdomain.Trip{}, tripdomain.ErrTripNotFound)

	_, err := uc.Execute(context.Background(), id)
	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("expected ErrTripNotFound, got %v", err)
	}
}
