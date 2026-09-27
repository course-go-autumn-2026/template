package v1

import (
	"github.com/imrtlDrifter/template/internal/service"
	"github.com/imrtlDrifter/template/pkg/database"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db       *database.Database
	services *service.Service
}

func NewHandler(db *database.Database, services *service.Service) *Handler {
	return &Handler{
		db:       db,
		services: services,
	}
}

func (h *Handler) Init(apiRouter *gin.RouterGroup) {
	v1 := apiRouter.Group("/v1")
	{
		h.initTripRoutes(v1)
	}
}
