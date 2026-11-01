package app

import (
	"encoding/json"
	"log"
	"net/http"

	api "github.com/makkssik/template/internal/generated"
)

func (i *Implementation) Ready(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK
	response := api.HealthResponse{Status: api.Ok}
	if err := i.checkupService.Check(r.Context()); err != nil {
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
