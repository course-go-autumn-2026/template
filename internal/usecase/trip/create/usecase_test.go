package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"

	tripdomain "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/usecase/trip/create"
)

func TestCreateSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{
		UserId:     uuid.New(),
		DriverId:   uuid.New(),
		StartPoint: tripdomain.Point{Latitude: 1, Longitude: 2},
		EndPoint:   tripdomain.Point{Latitude: 3, Longitude: 4},
		Price:      100,
	}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, v tripdomain.Trip) (tripdomain.Trip, error) {
		v.StartedAt = time.Now().UTC()
		return v, nil
	})
	history.EXPECT().Append(gomock.Any(), gomock.Any(), nil, tripdomain.StatusActive, gomock.Any()).Return(nil)

	out, err := uc.Execute(context.Background(), in, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Fresh {
		t.Fatal("expected fresh=true")
	}
	if out.Trip.Status != tripdomain.StatusActive {
		t.Fatalf("status = %v", out.Trip.Status)
	}
}

func TestCreateDriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{UserId: uuid.New(), DriverId: uuid.New(), Price: 10}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tripdomain.Trip{}, tripdomain.ErrDriverBusy)

	_, err := uc.Execute(context.Background(), in, nil)
	if !errors.Is(err, tripdomain.ErrDriverBusy) {
		t.Fatalf("expected ErrDriverBusy, got %v", err)
	}
}

func TestCreateRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{UserId: uuid.New(), DriverId: uuid.New(), Price: 10}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tripdomain.Trip{}, errors.New("db down"))

	_, err := uc.Execute(context.Background(), in, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateIdempotencyReplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{UserId: uuid.New(), DriverId: uuid.New(), Price: 10}
	key := uuid.New()
	stored := tripdomain.Trip{Id: uuid.New(), Status: tripdomain.StatusActive}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	idem.EXPECT().Claim(gomock.Any(), key, gomock.Any(), gomock.Any()).Return(false, nil)
	idem.EXPECT().Prune(gomock.Any(), key).Return(nil)
	idem.EXPECT().Replay(gomock.Any(), key, gomock.Any()).Return(stored, nil)

	out, err := uc.Execute(context.Background(), in, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Fresh {
		t.Fatal("expected fresh=false")
	}
	if out.Trip.Id != stored.Id {
		t.Fatal("trip mismatch")
	}
}

func TestCreateIdempotencyConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{UserId: uuid.New(), DriverId: uuid.New(), Price: 10}
	key := uuid.New()

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	idem.EXPECT().Claim(gomock.Any(), key, gomock.Any(), gomock.Any()).Return(false, nil)
	idem.EXPECT().Prune(gomock.Any(), key).Return(nil)
	idem.EXPECT().Replay(gomock.Any(), key, gomock.Any()).Return(tripdomain.Trip{}, tripdomain.ErrIdempotencyConflict)

	_, err := uc.Execute(context.Background(), in, &key)
	if !errors.Is(err, tripdomain.ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestCreateIdempotencyFresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	history := NewMockhistoryRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)

	uc := create.New(repo, tx, history, idem, time.Hour, nil)

	in := create.Input{UserId: uuid.New(), DriverId: uuid.New(), Price: 10}
	key := uuid.New()
	created := tripdomain.Trip{Id: uuid.New(), Status: tripdomain.StatusActive}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	idem.EXPECT().Claim(gomock.Any(), key, gomock.Any(), gomock.Any()).Return(true, nil)
	idem.EXPECT().Prune(gomock.Any(), key).Return(nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(created, nil)
	history.EXPECT().Append(gomock.Any(), created.Id, nil, tripdomain.StatusActive, gomock.Any()).Return(nil)
	idem.EXPECT().Save(gomock.Any(), key, created).Return(nil)

	out, err := uc.Execute(context.Background(), in, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Fresh {
		t.Fatal("expected fresh=true")
	}
}
