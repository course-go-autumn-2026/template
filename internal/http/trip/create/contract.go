package create

import (
	"context"

	"github.com/google/uuid"

	uc "github.com/murmyauuu/template/internal/usecase/trip/create"
)

type usecase interface {
	Execute(ctx context.Context, in uc.Input, key *uuid.UUID) (uc.Output, error)
}
