package finish_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	active := model.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), StartPoint: model.Coordinates{Latitude: 1, Longitude: 2}, EndPoint: model.Coordinates{Latitude: 3, Longitude: 4}, Price: 500, Status: string(trip.StatusActive), StartedAt: time.Now().UTC().Add(-time.Minute)}
	completed := active
	completed.Status = string(trip.StatusCompleted)
	setup := func(current model.Trip, getErr, updateErr, historyErr error) func(*testing.T, *gomock.Controller) *finish.Usecase {
		return func(t *testing.T, ctrl *gomock.Controller) *finish.Usecase {
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(current, getErr)
			if getErr == nil && current.Status == string(trip.StatusActive) {
				var updated model.Trip
				repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got model.Trip) error {
					require.NotNil(t, got.FinishedAt)
					expected := current
					expected.Status = string(trip.StatusCompleted)
					expected.FinishedAt = got.FinishedAt
					assert.Equal(t, expected, got)
					assert.False(t, got.FinishedAt.Before(got.StartedAt))
					updated = got
					return updateErr
				})
				if updateErr == nil {
					repo.EXPECT().AddStatusHistory(gomock.Any(), id, gomock.Any(), string(trip.StatusCompleted), "trip finished", gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, from *string, _ string, _ string, at time.Time) error {
						require.NotNil(t, from)
						assert.Equal(t, string(trip.StatusActive), *from)
						require.NotNil(t, updated.FinishedAt)
						assert.Equal(t, *updated.FinishedAt, at)
						return historyErr
					})
				}
			}
			return finish.New(repo, tx)
		}
	}
	tests := []struct {
		name    string
		setup   func(*testing.T, *gomock.Controller) *finish.Usecase
		wantErr error
	}{
		{"success", setup(active, nil, nil, nil), nil},
		{"already completed", setup(completed, nil, nil, nil), trip.ErrTripCompleted},
		{"not found", setup(model.Trip{}, trip.ErrTripNotFound, nil, nil), trip.ErrTripNotFound},
		{"update error", setup(active, nil, assert.AnError, nil), assert.AnError},
		{"history error", setup(active, nil, nil, assert.AnError), assert.AnError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := tt.setup(t, gomock.NewController(t))
			got, err := uc.Execute(context.Background(), id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got.FinishedAt)
			expected := active
			expected.Status = string(trip.StatusCompleted)
			expected.FinishedAt = got.FinishedAt
			assert.Equal(t, expected, got)
		})
	}
}
