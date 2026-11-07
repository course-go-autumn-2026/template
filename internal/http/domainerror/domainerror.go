package domainerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	api "github.com/XrystikLK/template/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	if errors.Is(err, domainTrip.ErrTripNotFound) {
		code := "trip_not_found"
		detail := "Trip was not found"
		return http.StatusNotFound, api.Problem{
			Code:   code,
			Status: http.StatusNotFound,
			Title:  "Trip not found",
			Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
			Detail: &detail,
		}
	}

	if errors.Is(err, domainTrip.ErrTripCompleted) {
		code := "trip_completed"
		detail := "Operation is not allowed for a completed trip"
		return http.StatusConflict, api.Problem{
			Code:   code,
			Status: http.StatusConflict,
			Title:  "Trip completed",
			Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
			Detail: &detail,
		}
	}

	if errors.Is(err, domainTrip.ErrDriverBusy) {
		code := "driver_busy"
		detail := "Driver already has an active trip"
		return http.StatusConflict, api.Problem{
			Code:   code,
			Status: http.StatusConflict,
			Title:  "Driver busy",
			Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
			Detail: &detail,
		}
	}

	var valErr *domainTrip.ValidationError
	if errors.As(err, &valErr) {
		code := "invalid_request"
		detail := valErr.Error()
		return http.StatusBadRequest, api.Problem{
			Code:   code,
			Status: http.StatusBadRequest,
			Title:  "Invalid request",
			Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
			Detail: &detail,
		}
	}

	code := "internal_error"
	return http.StatusInternalServerError, api.Problem{
		Code:   code,
		Status: http.StatusInternalServerError,
		Title:  "Internal Server Error",
		Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
	}
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if r != nil {
		path := r.URL.Path
		body.Instance = &path

		if status >= http.StatusInternalServerError {
			slog.ErrorContext(r.Context(), "internal server error",
				slog.String("error", err.Error()),
				slog.String("path", path),
			)

			span := trace.SpanFromContext(r.Context())
			if span.IsRecording() {
				span.SetStatus(codes.Error, err.Error())
				span.SetAttributes(attribute.String("error.type", "internal_error"))
			}
		}
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
