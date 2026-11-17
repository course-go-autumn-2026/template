package listpositions_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/http/trip/listpositions"
)

func TestEmptyRouteAndMissingTrip(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"empty", nil, 200}, {"missing", trip.ErrTripNotFound, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewMockusecase(gomock.NewController(t))
			id := uuid.New()
			usecase.EXPECT().Execute(gomock.Any(), id).Return(nil, tc.err)
			router := chi.NewRouter()
			router.Get("/{tripId}", listpositions.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))).ServeHTTP)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/"+id.String(), nil))

			if response.Code != tc.status {
				t.Fatalf("status: %d", response.Code)
			}

			if tc.err == nil && !strings.Contains(response.Body.String(), `"positions":[]`) {
				t.Fatal("empty route must be an array")
			}
		})
	}
}
