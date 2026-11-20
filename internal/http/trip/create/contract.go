//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"

	usecase "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

type (
	tripCreator interface {
		Execute(ctx context.Context, in usecase.Input) (usecase.Result, error)
	}

	metrics interface {
		IncCreated(ctx context.Context)
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}
)
