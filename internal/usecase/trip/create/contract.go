//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/xotapy-main/courseGo/internal/model"
)

type (
	tripRepository interface {
		Save(ctx context.Context, trip *model.Trip) error
		InsertHistory(ctx context.Context, record *model.TripStatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
