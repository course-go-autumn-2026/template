package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

type UseCase struct{ repo tripRepository }

func New(repo tripRepository) *UseCase { return &UseCase{repo: repo} }
func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return u.repo.Get(ctx, id)
}
