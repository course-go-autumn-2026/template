package v1

import (
	api "github.com/imrtlDrifter/template/internal/generated"
	"github.com/imrtlDrifter/template/internal/service"
	"github.com/imrtlDrifter/template/pkg/database"
	httpres "github.com/imrtlDrifter/template/pkg/http_res"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db       *database.Database
	services *service.Service
}

var _ api.ServerInterface = (*Handler)(nil)

func NewHandler(db *database.Database, services *service.Service) *Handler {
	return &Handler{
		db:       db,
		services: services,
	}
}

func (h *Handler) Init(router gin.IRouter) {
	api.RegisterHandlersWithOptions(router, h, api.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			httpres.NewError(c, statusCode, err)
		},
	})
}
