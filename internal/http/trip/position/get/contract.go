//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) ([]trip.Position, error)
}
