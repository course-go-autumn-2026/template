package finish

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

func TestExecuteFinish(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	id := uuid.New()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{ID: id, Status: domain.StatusActive}, nil)
	repo.EXPECT().Finish(gomock.Any(), id).Return(domain.Trip{ID: id, Status: domain.StatusCompleted}, nil)
	repo.EXPECT().AddStatusHistory(gomock.Any(), id, gomock.Any(), domain.StatusCompleted, "trip finished").Return(nil)
	trip, err := New(repo, tx, nil).Execute(context.Background(), id)
	if err != nil || trip.Status != domain.StatusCompleted {
		t.Fatalf("Execute() = %+v, %v", trip, err)
	}
}

func TestExecuteRepeatedFinish(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	id := uuid.New()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{ID: id, Status: domain.StatusCompleted}, nil)
	_, err := New(repo, tx, nil).Execute(context.Background(), id)
	if !errors.Is(err, domain.ErrTripCompleted) {
		t.Fatalf("Execute() error = %v", err)
	}
}
