package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/http/domainerror"
)

func JSON(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, contentType string, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		logger.ErrorContext(r.Context(), "encode HTTP response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)

	if _, err := w.Write(append(body, '\n')); err != nil {
		logger.ErrorContext(r.Context(), "write HTTP response", "error", err)
	}
}

func Error(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, body := domainerror.Problem(err)
	instance := r.URL.Path
	body.Instance = &instance

	if status >= 500 {
		logger.ErrorContext(r.Context(), "request failed", "error", err)
	}

	JSON(w, r, logger, status, "application/problem+json", body)
}

func Invalid(w http.ResponseWriter, r *http.Request, logger *slog.Logger, message string) {
	Error(w, r, logger, &trip.ValidationError{Field: "request", Message: message})
}
