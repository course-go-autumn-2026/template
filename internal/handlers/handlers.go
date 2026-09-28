package handlers

import (
	handlerv1 "github.com/imrtlDrifter/template/internal/handlers/v1"
	"github.com/imrtlDrifter/template/internal/service"
	"github.com/imrtlDrifter/template/pkg/database"
	"github.com/imrtlDrifter/template/pkg/logger"

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

func (h *Handler) Init() *gin.Engine {
	gin.DisableConsoleColor()
	gin.EnableJsonDecoderDisallowUnknownFields()
	router := gin.New()
	router.HandleMethodNotAllowed = true

	router.Use(
		logger.WithRequestID(),
		logger.GinMiddleware(),
		gin.Recovery(),
	)

	h.initAPI(router)

	return router
}

func (h *Handler) initAPI(router *gin.Engine) {
	handlerV1 := handlerv1.NewHandler(h.db, h.services)
	handlerV1.Init(router)
}
