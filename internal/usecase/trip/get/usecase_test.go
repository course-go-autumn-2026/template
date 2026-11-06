package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
	"github.com/toooriii/tori/internal/usecase/trip/get"
)

func TestUseCase_Do_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	id := uuid.New()
	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{ID: id}, nil)

	uc := get.New(repo)
	got, err := uc.Do(context.Background(), id)
	if err != nil || got.ID != id {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestUseCase_Do_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	id := uuid.New()
	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	uc := get.New(repo)
	_, err := uc.Do(context.Background(), id)
	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("err = %v", err)
	}
}
