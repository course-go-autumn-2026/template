package v1

import (
	"net/http"

	api "github.com/imrtlDrifter/template/internal/generated"
	"github.com/imrtlDrifter/template/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(c *gin.Context) {
	if err := h.db.CheckReady(c.Request.Context()); err != nil {
		logger.ErrorCtx(c, err, "readiness_check_failed")
		c.JSON(http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	c.JSON(http.StatusOK, api.HealthResponse{Status: api.Ok})
}
