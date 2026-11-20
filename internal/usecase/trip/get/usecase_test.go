package get_test

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
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	expected := model.Trip{ID: id, UserID: uuid.New(), Price: 500, Status: string(trip.StatusActive)}
	tests := []struct {
		name    string
		setup   func(*gomock.Controller) *get.Usecase
		want    model.Trip
		wantErr error
	}{
		{name: "found", setup: func(ctrl *gomock.Controller) *get.Usecase {
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().Get(gomock.Any(), id).Return(expected, nil)
			return get.New(repo, time.Second)
		}, want: expected},
		{name: "not found", setup: func(ctrl *gomock.Controller) *get.Usecase {
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)
			return get.New(repo, time.Second)
		}, wantErr: trip.ErrTripNotFound},
		{name: "repository error", setup: func(ctrl *gomock.Controller) *get.Usecase {
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, assert.AnError)
			return get.New(repo, time.Second)
		}, wantErr: assert.AnError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := tt.setup(gomock.NewController(t))
			got, err := uc.Execute(context.Background(), id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
