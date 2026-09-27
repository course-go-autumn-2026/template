package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/domainerrors"
	"github.com/imrtlDrifter/template/internal/models"
	httpres "github.com/imrtlDrifter/template/pkg/http_res"
	"github.com/imrtlDrifter/template/pkg/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initTripRoutes(router *gin.RouterGroup) {
	router.POST("/trips", h.CreateTrip)
	router.GET("/trips/:tripId", h.GetTrip)
	router.POST("/trips/:tripId/finish", h.FinishTrip)
}

func (h *Handler) CreateTrip(c *gin.Context) {
	var request models.CreateTripRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		if errors.Is(err, io.EOF) {
			httpres.HandleDomainError(c, domainerrors.ErrEmptyRequestBody)
			return
		}
		httpres.NewError(c, http.StatusBadRequest, err)
		return
	}

	var idempotencyKey *uuid.UUID
	header := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if header != "" {
		key, err := uuid.Parse(header)
		if err != nil {
			httpres.NewError(c, http.StatusBadRequest, err)
			return
		}
		idempotencyKey = &key
	}

	trip, created, err := h.services.Trip.Create(c.Request.Context(), &request, idempotencyKey)
	if err != nil {
		httpres.HandleDomainError(c, domainerrors.WrapValidationError(err))
		return
	}

	if created {
		c.Header("Location", fmt.Sprintf("%s/v1/trips/%s", utils.APIBasePath, trip.ID))
		c.JSON(http.StatusCreated, trip)
		return
	}

	c.JSON(http.StatusOK, trip)
}

func (h *Handler) GetTrip(c *gin.Context) {
	tripID, err := uuid.Parse(c.Param("tripId"))
	if err != nil {
		httpres.NewError(c, http.StatusBadRequest, err)
		return
	}

	trip, err := h.services.Trip.GetByID(c.Request.Context(), tripID)
	if err != nil {
		httpres.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, trip)
}

func (h *Handler) FinishTrip(c *gin.Context) {
	tripID, err := uuid.Parse(c.Param("tripId"))
	if err != nil {
		httpres.NewError(c, http.StatusBadRequest, err)
		return
	}

	trip, err := h.services.Trip.Finish(c.Request.Context(), tripID)
	if err != nil {
		httpres.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, trip)
}
