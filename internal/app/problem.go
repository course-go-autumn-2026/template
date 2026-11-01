package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/makkssik/template/internal/domain/trips"
	api "github.com/makkssik/template/internal/generated"
)

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var businessErr *trips.Error
	if errors.As(err, &businessErr) && businessErr != nil {
		status := statusByKind(businessErr.Kind)
		if status != http.StatusInternalServerError {
			writeProblem(w, r, status, businessErr.Code, http.StatusText(status), businessErr.Message)
			return
		}
	}

	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
}

func statusByKind(kind trips.Kind) int {
	switch kind {
	case trips.KindNotFound:
		return http.StatusNotFound
	case trips.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.Printf("write problem response: %v", err)
	}
}
