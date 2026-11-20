package positions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
)

type txManager interface {
	Do(context.Context, func(context.Context) error) error
}
type canceller interface {
	Cancel(context.Context, uuid.UUID)
}
type Handler struct {
	db      *pgxpool.Pool
	tx      txManager
	cancel  canceller
	timeout time.Duration
	saved   metric.Int64Counter
	logger  *slog.Logger
}

func New(db *pgxpool.Pool, tx txManager, c canceller, timeout time.Duration, m metric.Meter, logger *slog.Logger) (*Handler, error) {
	counter, err := m.Int64Counter("tripgo.positions.saved")
	if err != nil {
		return nil, err
	}
	return &Handler{db, tx, c, timeout, counter, logger}, nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var body struct {
		Coordinates *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"coordinates"`
		RecordedAt *time.Time `json:"recorded_at"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	err := dec.Decode(&body)
	if err == nil {
		var extra any
		if e := dec.Decode(&extra); e != io.EOF {
			err = errors.New("invalid JSON")
		}
	}
	if err != nil || body.Coordinates == nil || body.Coordinates.Latitude == nil || body.Coordinates.Longitude == nil || body.RecordedAt == nil || body.RecordedAt.IsZero() || body.RecordedAt.After(time.Now().Add(time.Minute)) || *body.Coordinates.Latitude < -90 || *body.Coordinates.Latitude > 90 || *body.Coordinates.Longitude < -180 || *body.Coordinates.Longitude > 180 {
		httputil.WriteProblem(w, r, 400, httputil.Problem("invalid_request", "Invalid request", "Invalid coordinates or recorded_at", 400))
		return
	}
	pos := api.TripPosition{TripId: id, Coordinates: api.Coordinates{Latitude: *body.Coordinates.Latitude, Longitude: *body.Coordinates.Longitude}, RecordedAt: *body.RecordedAt}
	err = h.tx.Do(r.Context(), func(ctx context.Context) error {
		db := postgres.Executor(ctx, h.db)
		var status string
		err := db.QueryRow(ctx, `SELECT status FROM trips WHERE id=$1 FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return trip.ErrTripNotFound
		}
		if err != nil {
			return fmt.Errorf("lock trip: %w", err)
		}
		if status != "active" {
			return trip.ErrTripCompleted
		}
		err = db.QueryRow(ctx, `INSERT INTO trip_positions(trip_id,latitude,longitude,recorded_at) VALUES ($1,$2,$3,$4) RETURNING id`, id, pos.Coordinates.Latitude, pos.Coordinates.Longitude, pos.RecordedAt).Scan(&pos.Id)
		if err != nil {
			return fmt.Errorf("insert position: %w", err)
		}
		_, err = db.Exec(ctx, `UPDATE trips SET last_position_at=GREATEST(last_position_at,$2) WHERE id=$1`, id, pos.RecordedAt)
		if err != nil {
			return fmt.Errorf("update position time: %w", err)
		}
		return nil
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.cancel.Cancel(r.Context(), id)
	h.saved.Add(r.Context(), 1)
	httputil.WriteJSON(w, 201, pos)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	var exists bool
	if err := h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM trips WHERE id=$1)`, id).Scan(&exists); err != nil {
		h.writeError(w, r, err)
		return
	}
	if !exists {
		h.writeError(w, r, trip.ErrTripNotFound)
		return
	}
	rows, err := h.db.Query(ctx, `SELECT id, latitude, longitude, recorded_at FROM trip_positions WHERE trip_id=$1 ORDER BY recorded_at,id`, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer rows.Close()
	result := make([]api.TripPosition, 0)
	for rows.Next() {
		p := api.TripPosition{TripId: id}
		if err = rows.Scan(&p.Id, &p.Coordinates.Latitude, &p.Coordinates.Longitude, &p.RecordedAt); err != nil {
			h.writeError(w, r, err)
			return
		}
		result = append(result, p)
	}
	if err = rows.Err(); err != nil {
		h.writeError(w, r, err)
		return
	}
	httputil.WriteJSON(w, 200, api.TripPositionsResponse{Positions: result})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, problem := domainerror.Problem(err)
	if status == 500 {
		h.logger.ErrorContext(r.Context(), "positions request failed", "error", err)
	}
	httputil.WriteProblem(w, r, status, problem)
}
