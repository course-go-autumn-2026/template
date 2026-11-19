//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/xotapy-main/courseGo/internal/model"
	ucposcreate "github.com/xotapy-main/courseGo/internal/usecase/position/create"
)

type usecase interface {
	Execute(ctx context.Context, req ucposcreate.Request) (*model.Position, error)
}
