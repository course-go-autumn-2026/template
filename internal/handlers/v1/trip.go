package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/imrtlDrifter/template/internal/domainerrors"
	api "github.com/imrtlDrifter/template/internal/generated"
	"github.com/imrtlDrifter/template/internal/models"
	httpres "github.com/imrtlDrifter/template/pkg/http_res"
	"github.com/imrtlDrifter/template/pkg/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateTrip(c *gin.Context, params api.CreateTripParams) {
	var request api.CreateTripJSONRequestBody
	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		if errors.Is(err, io.EOF) {
			httpres.HandleDomainError(c, domainerrors.ErrEmptyRequestBody)
			return
		}
		httpres.NewError(c, http.StatusBadRequest, err)
		return
	}

	body := c.MustGet(gin.BodyBytesKey).([]byte)
	if err := utils.ValidateCreateTripRequiredFields(body); err != nil {
		httpres.HandleDomainError(c, domainerrors.WrapValidationError(err))
		return
	}

	trip, created, err := h.services.Trip.Create(c.Request.Context(), models.CreateTrip(request), params.IdempotencyKey)
	if err != nil {
		httpres.HandleDomainError(c, domainerrors.WrapValidationError(err))
		return
	}

	if created {
		c.Header("Location", fmt.Sprintf("%s/v1/trips/%s", utils.APIBasePath, trip.ID))
		c.JSON(http.StatusCreated, models.TripResponse(trip))
		return
	}

	c.JSON(http.StatusOK, models.TripResponse(trip))
}

func (h *Handler) GetTrip(c *gin.Context, tripID api.TripId) {
	trip, err := h.services.Trip.GetByID(c.Request.Context(), tripID)
	if err != nil {
		httpres.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TripResponse(trip))
}

func (h *Handler) FinishTrip(c *gin.Context, tripID api.TripId) {
	trip, err := h.services.Trip.Finish(c.Request.Context(), tripID)
	if err != nil {
		httpres.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TripResponse(trip))
}
