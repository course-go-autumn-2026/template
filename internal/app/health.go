package app

import (
	"encoding/json"
	"log"
	"net/http"

	api "github.com/makkssik/template/internal/generated"
)

func (i *Implementation) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok}); err != nil {
		log.Printf("write health response: %v", err)
	}
}
