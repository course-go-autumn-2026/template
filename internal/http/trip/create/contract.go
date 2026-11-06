//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package create

import (
	"context"

	usecase "github.com/toooriii/tori/internal/usecase/trip/create"
)

type UseCase interface {
	Do(ctx context.Context, in usecase.Input) (usecase.Output, error)
}
