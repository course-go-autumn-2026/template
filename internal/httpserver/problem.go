package httpserver

import (
	"encoding/json"
	"net/http"

	api "github.com/Yolanesso/avito-labs/internal/generated"
)

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	problemType string,
	title string,
	detail string,
	code string,
) {
	instance := r.URL.Path

	problem := api.Problem{
		Type:     problemType,
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set(
		"Content-Type",
		"application/problem+json",
	)

	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(problem)
	if err != nil {
		return
	}
}
