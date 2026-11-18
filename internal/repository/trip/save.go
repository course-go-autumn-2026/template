package trip

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/infra/postgres/transaction"
	"github.com/XrystikLK/template/internal/model"
)

func (r *Repository) Save(ctx context.Context, tr model.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.sq.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
			"created_at", "updated_at",
		).
		Values(
			tr.ID, tr.UserID, tr.DriverID,
			tr.StartPoint.Latitude, tr.StartPoint.Longitude,
			tr.EndPoint.Latitude, tr.EndPoint.Longitude,
			tr.Price, string(tr.Status), tr.StartedAt,
			tr.CreatedAt, tr.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return err
	}

	var exec executor = transaction.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domainTrip.ErrDriverBusy
		}
		return err
	}

	return nil
}

func (r *Repository) AddStatusHistory(ctx context.Context, history model.StatusHistory) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var fromStatus *string
	if history.FromStatus != nil {
		s := string(*history.FromStatus)
		fromStatus = &s
	}

	query, args, err := r.sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(history.TripID, fromStatus, string(history.ToStatus), history.Reason, history.ChangedAt).
		ToSql()
	if err != nil {
		return err
	}

	var exec executor = transaction.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	return err
}

func (r *Repository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	now := time.Now()
	query, args, err := r.sq.Update("trips").
		Set("status", string(model.StatusCompleted)).
		Set("finished_at", finishedAt).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id, "status": string(model.StatusActive)}).
		Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, err
	}

	var exec executor = transaction.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	t, err := scanTrip(row)
	if err == nil {
		return t, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		checkQuery, checkArgs, checkErr := r.sq.Select("status").From("trips").Where(squirrel.Eq{"id": id}).ToSql()
		if checkErr != nil {
			return nil, checkErr
		}

		var currentStatus string
		checkErr = exec.QueryRow(ctx, checkQuery, checkArgs...).Scan(&currentStatus)
		if errors.Is(checkErr, pgx.ErrNoRows) {
			return nil, domainTrip.ErrTripNotFound
		}
		if currentStatus == string(model.StatusCompleted) {
			return nil, domainTrip.ErrTripCompleted
		}
		if checkErr != nil {
			return nil, checkErr
		}
	}

	return nil, err
}
