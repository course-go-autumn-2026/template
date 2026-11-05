package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	tripDomain "github.com/ggwpfu9/courseAvito/internal/domain/trip"
	appmodel "github.com/ggwpfu9/courseAvito/internal/model" 
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func GetExecutor(ctx context.Context, pool *pgxpool.Pool) interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
} {
	type txKey struct{}
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

func (r *Repo) Save(ctx context.Context, trip appmodel.Trip) error {
	exec := GetExecutor(ctx, r.pool)

	query, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_lat", "start_lon", "end_lat", "end_lon", "price", "status", "started_at").
		Values(trip.ID, trip.UserID, trip.DriverID, trip.StartPoint.Latitude, trip.StartPoint.Longitude, trip.EndPoint.Latitude, trip.EndPoint.Longitude, trip.Price, trip.Status, trip.StartedAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return tripDomain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}

	historyQuery, historyArgs, _ := sq.Insert("trip_status_history").
		Columns("trip_id", "status", "changed_at").
		Values(trip.ID, trip.Status, time.Now()).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	_, err = exec.Exec(ctx, historyQuery, historyArgs...)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return nil
}

func (r *Repo) GetByID(ctx context.Context, id string) (*appmodel.Trip, error) {
	exec := GetExecutor(ctx, r.pool)

	query, args, _ := sq.Select("id", "user_id", "driver_id", "start_lat", "start_lon", "end_lat", "end_lon", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	var t appmodel.Trip
	err := exec.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID, &t.StartPoint.Latitude, &t.StartPoint.Longitude, &t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tripDomain.ErrTripNotFound
		}
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return &t, nil
}

func (r *Repo) Update(ctx context.Context, trip appmodel.Trip) error {
	exec := GetExecutor(ctx, r.pool)
	query, args, _ := sq.Update("trips").
		Set("status", trip.Status).
		Set("finished_at", trip.FinishedAt).
		Where(sq.Eq{"id": trip.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	_, err := exec.Exec(ctx, query, args...)
	return err
}