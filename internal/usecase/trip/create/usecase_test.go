package create

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

func TestExecuteCreatesTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	ctx := context.Background()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, trip domain.Trip) (domain.Trip, error) {
		if trip.Status != domain.StatusActive || trip.ID == uuid.Nil {
			t.Fatalf("invalid new trip: %+v", trip)
		}
		return trip, nil
	})
	repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, domain.StatusActive, "trip created").Return(nil)
	result, err := New(repo, tx, nil).Execute(ctx, Request{UserID: uuid.New(), DriverID: uuid.New(), Price: 100})
	if err != nil || result.Replayed || result.Trip.ID == uuid.Nil {
		t.Fatalf("Execute() = %+v, %v", result, err)
	}
}

func TestExecuteDriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Trip{}, domain.ErrDriverBusy)
	_, err := New(repo, tx, nil).Execute(context.Background(), Request{})
	if !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteIdempotentReplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	key, id := uuid.New(), uuid.New()
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().ReserveIdempotencyKey(gomock.Any(), key, "hash").Return(domain.IdempotencyRecord{RequestHash: "hash", TripID: id}, false, nil)
	repo.EXPECT().Get(gomock.Any(), id).Return(domain.Trip{ID: id}, nil)
	result, err := New(repo, tx, nil).Execute(context.Background(), Request{IdempotencyKey: &key, RequestHash: "hash"})
	if err != nil || !result.Replayed || result.Trip.ID != id {
		t.Fatalf("Execute() = %+v, %v", result, err)
	}
}

func TestExecuteRollsBackOnHistoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo, tx := NewMocktripRepository(ctrl), NewMocktxManager(ctrl)
	want := errors.New("history failed")
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, trip domain.Trip) (domain.Trip, error) { return trip, nil })
	repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, domain.StatusActive, "trip created").Return(want)
	_, err := New(repo, tx, nil).Execute(context.Background(), Request{})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteBusinessValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		req   Request
		field string
	}{
		{"latitude", Request{StartPoint: domain.Coordinates{Latitude: 91}}, "start_point.latitude"},
		{"longitude", Request{EndPoint: domain.Coordinates{Longitude: -181}}, "end_point.longitude"},
		{"price", Request{Price: -1}, "price"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			_, err := New(NewMocktripRepository(ctrl), NewMocktxManager(ctrl), nil).Execute(context.Background(), tc.req)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
