package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
	"github.com/toooriii/tori/internal/usecase/trip/finish"
)

func passthrough(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func TestUseCase_Do_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	id := uuid.New()
	now := time.Now().UTC()

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(passthrough)
	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{
		ID: id, Status: trip.StatusActive,
	}, nil)
	repo.EXPECT().Finish(gomock.Any(), id, now).Return(model.Trip{
		ID: id, Status: trip.StatusCompleted,
	}, nil)
	repo.EXPECT().AddHistory(gomock.Any(), id, gomock.Any(), "completed", "trip finished").Return(nil)

	uc := finish.New(repo, tx)
	out, err := uc.Do(context.Background(), id, now)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if out.Status != trip.StatusCompleted {
		t.Fatalf("status = %v", out.Status)
	}
}

func TestUseCase_Do_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	id := uuid.New()
	now := time.Now().UTC()

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(passthrough)
	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{
		ID: id, Status: trip.StatusCompleted,
	}, nil)

	uc := finish.New(repo, tx)
	_, err := uc.Do(context.Background(), id, now)
	if !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("err = %v", err)
	}
}

func TestUseCase_Do_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	id := uuid.New()
	now := time.Now().UTC()

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(passthrough)
	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	uc := finish.New(repo, tx)
	_, err := uc.Do(context.Background(), id, now)
	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("err = %v", err)
	}
}
