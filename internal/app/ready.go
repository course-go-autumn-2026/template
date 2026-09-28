package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	api "github.com/makkssik/template/internal/generated"
)

func (i *Implementation) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), i.queryTimeout)
	defer cancel()

	statusCode := http.StatusOK
	response := api.HealthResponse{Status: api.Ok}
	if err := i.pool.Ping(ctx); err != nil {
		log.Printf("check database readiness: %v", err)
		statusCode = http.StatusServiceUnavailable
		response.Status = api.Unavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write readiness response: %v", err)
	}
}
