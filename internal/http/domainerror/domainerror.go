package domainerror

import (
	"encoding/json"
	"net/http"

	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror/strategy"
)

var (
	registeredStrategies = strategy.DefaultStrategies()
	fallbackStrategy     = strategy.NewInternalStrategy()
)

func Problem(err error) (int, api.Problem) {
	for _, s := range registeredStrategies {
		if status, p, ok := s.Handle(err); ok {
			return status, p
		}
	}

	status, p, _ := fallbackStrategy.Handle(err)
	return status, p
}

func Write(w http.ResponseWriter, err error) {
	status, body := Problem(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
