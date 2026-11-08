//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/XrystikLK/template/internal/model"
	createUC "github.com/XrystikLK/template/internal/usecase/trip/create"
)

type (
	useCase interface {
		Execute(ctx context.Context, req createUC.Request) (*model.Trip, error)
	}
)
