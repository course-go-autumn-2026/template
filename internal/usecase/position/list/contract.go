//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type positionRepository interface {
	ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.Position, error)
}
