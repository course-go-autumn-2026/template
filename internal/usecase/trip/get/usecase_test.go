package get

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

func TestExecuteMissingTrip(t *testing.T) {
	repo := NewMocktripRepository(gomock.NewController(t))
	id := uuid.New()
	repo.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{}, domain.ErrTripNotFound)
	_, err := New(repo).Execute(context.Background(), id)
	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("Execute() error = %v", err)
	}
}
