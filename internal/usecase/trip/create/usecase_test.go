package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

func TestUsecase_Execute(t *testing.T) {
	key := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	existing := model.Trip{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Status: string(trip.StatusActive)}
	hash := []byte("request-hash")
	writeErr := errors.New("write conflict")
	commitErr := errors.New("commit failed")
	input := create.Input{
		UserID:     uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		DriverID:   uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		StartPoint: trip.Coordinates{Latitude: 55.7, Longitude: 37.6},
		EndPoint:   trip.Coordinates{Latitude: 55.8, Longitude: 37.7},
		Price:      500,
	}

	tests := []struct {
		name                          string
		withKey, replay, conflict     bool
		createErr, completeErr, txErr error
		wantErr                       error
		wantCreated                   bool
	}{
		{name: "create", wantCreated: true},
		{name: "create with idempotency key", withKey: true, wantCreated: true},
		{name: "idempotent replay", withKey: true, replay: true},
		{name: "idempotency conflict", withKey: true, replay: true, conflict: true, wantErr: trip.ErrIdempotencyConflict},
		{name: "driver busy", createErr: trip.ErrDriverBusy, wantErr: trip.ErrDriverBusy},
		{name: "repository error", createErr: writeErr, wantErr: writeErr},
		{name: "complete key error", withKey: true, completeErr: writeErr, wantErr: writeErr},
		{name: "transaction commit error", txErr: commitErr, wantErr: commitErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				if err := fn(ctx); err != nil {
					return err
				}
				return tt.txErr
			})

			in := input
			if tt.withKey {
				in.IdempotencyKey, in.RequestHash = &key, hash
				existingHash := hash
				if tt.conflict {
					existingHash = []byte("different-hash")
				}
				repo.EXPECT().ClaimIdempotencyKey(gomock.Any(), key, hash).Return(!tt.replay, existingHash, existing.ID, nil)
			}

			var saved model.Trip
			if tt.replay && !tt.conflict {
				repo.EXPECT().Get(gomock.Any(), existing.ID).Return(existing, nil)
			} else if !tt.replay {
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got model.Trip) error {
					saved = got
					assert.NotEqual(t, uuid.Nil, got.ID)
					assert.Equal(t, in.UserID, got.UserID)
					assert.Equal(t, in.DriverID, got.DriverID)
					assert.Equal(t, model.Coordinates(in.StartPoint), got.StartPoint)
					assert.Equal(t, model.Coordinates(in.EndPoint), got.EndPoint)
					assert.Equal(t, in.Price, got.Price)
					assert.Equal(t, string(trip.StatusActive), got.Status)
					assert.False(t, got.StartedAt.IsZero())
					assert.Nil(t, got.FinishedAt)
					return tt.createErr
				})
				if tt.createErr == nil {
					repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, string(trip.StatusActive), "trip created", gomock.Any()).DoAndReturn(
						func(_ context.Context, id uuid.UUID, _ *string, _, _ string, at time.Time) error {
							assert.Equal(t, saved.ID, id)
							assert.Equal(t, saved.StartedAt, at)
							return nil
						},
					)
					if tt.withKey {
						repo.EXPECT().CompleteIdempotencyKey(gomock.Any(), key, gomock.Any()).DoAndReturn(func(_ context.Context, _, id uuid.UUID) error {
							assert.Equal(t, saved.ID, id)
							return tt.completeErr
						})
					}
				}
			}

			result, err := create.New(repo, tx).Execute(context.Background(), in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, create.Result{}, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, result.Created)
			if tt.replay {
				assert.Equal(t, existing, result.Trip)
			} else {
				assert.Equal(t, saved, result.Trip)
			}
		})
	}
}
