package create_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
	"github.com/toooriii/tori/internal/usecase/trip/create"
)

func validInput() create.Input {
	return create.Input{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Start:    model.Point{Latitude: 59.9, Longitude: 30.3},
		End:      model.Point{Latitude: 59.8, Longitude: 30.4},
		Price:    100,
		RawBody:  []byte(`{"user_id":"x"}`),
		Now:      time.Now(),
	}
}

func txPassthrough() func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
}

func TestUseCase_Do_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(txPassthrough())
	trips.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	trips.EXPECT().AddHistory(gomock.Any(), gomock.Any(), nil, "active", "trip created").Return(nil)

	uc := create.New(trips, idem, tx)
	out, err := uc.Do(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trip.Status != trip.StatusActive {
		t.Fatalf("status = %v, want active", out.Trip.Status)
	}
	if out.AlreadyExists {
		t.Fatalf("AlreadyExists should be false on first create")
	}
}

func TestUseCase_Do_DriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(txPassthrough())
	trips.EXPECT().Save(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

	uc := create.New(trips, idem, tx)
	_, err := uc.Do(context.Background(), validInput())
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("err = %v, want ErrDriverBusy", err)
	}
}

func TestUseCase_Do_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := create.New(NewMocktripRepository(ctrl), NewMockidempotencyRepository(ctrl), NewMocktxManager(ctrl))

	in := validInput()
	in.Price = -1
	_, err := uc.Do(context.Background(), in)
	var ve *trip.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "price" {
		t.Fatalf("field = %q, want price", ve.Field)
	}
}

func TestUseCase_Do_IdempotencyRepeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	key := uuid.New()
	in := validInput()
	in.Key = &key
	hash, err := hashOf(t, in.RawBody)
	if err != nil {
		t.Fatalf("hashOf: %v", err)
	}

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).Return(trip.ErrKeyAlreadyExists)
	idem.EXPECT().Get(gomock.Any(), key).Return(model.IdempotencyRecord{
		Key: key, RequestHash: hash, TripID: uuid.New(),
	}, nil)
	trips.EXPECT().Get(gomock.Any(), gomock.Any()).Return(model.Trip{
		ID: uuid.New(), Status: trip.StatusActive,
	}, nil)

	uc := create.New(trips, idem, tx)
	out, err := uc.Do(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.AlreadyExists {
		t.Fatalf("AlreadyExists should be true on repeat")
	}
}

func TestUseCase_Do_IdempotencyConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	idem := NewMockidempotencyRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	key := uuid.New()
	in := validInput()
	in.Key = &key

	tx.EXPECT().Do(gomock.Any(), gomock.Any()).Return(trip.ErrKeyAlreadyExists)
	idem.EXPECT().Get(gomock.Any(), key).Return(model.IdempotencyRecord{
		Key: key, RequestHash: "different-hash",
	}, nil)

	uc := create.New(trips, idem, tx)
	_, err := uc.Do(context.Background(), in)
	if !errors.Is(err, trip.ErrIdempotencyConflict) {
		t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
	}
}

func hashOf(t *testing.T, raw []byte) (string, error) {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	canon, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}
