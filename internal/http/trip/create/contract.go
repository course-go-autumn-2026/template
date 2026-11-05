//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/xotapy-main/courseGo/internal/model"
	uccreate "github.com/xotapy-main/courseGo/internal/usecase/trip/create"
)

type usecase interface {
	Execute(ctx context.Context, req uccreate.Request) (*model.Trip, error)
}
