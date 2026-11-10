package http

import (
	"encoding/json"
	"net/http"

	"github.com/murmyauuu/template/api"
)

func WriteJSON(w http.ResponseWriter, status int, contentType string, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteProblem(w http.ResponseWriter, status int, body api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
