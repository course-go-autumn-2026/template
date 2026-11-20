//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, data model.CreatePosition) (*trip.Position, error)
}
