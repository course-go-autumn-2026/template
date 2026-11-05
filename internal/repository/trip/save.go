package trip

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/model"
)

func (r *RepositoryImpl) Save(ctx context.Context, t *model.Trip) error {
	query, args, err := r.sb.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
			"finished_at", "last_position_at",
		).
		Values(
			t.ID, t.UserID, t.DriverID,
			t.StartPoint.Latitude, t.StartPoint.Longitude,
			t.EndPoint.Latitude, t.EndPoint.Longitude,
			t.Price, string(t.Status), t.StartedAt,
			t.FinishedAt, t.LastPositionAt,
		).
		ToSql()
	if err != nil {
		return err
	}

	exec := transaction.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return domaintrip.ErrDriverBusy
		}
		return err
	}

	return nil
}

func (r *RepositoryImpl) InsertHistory(ctx context.Context, record *model.TripStatusHistory) error {
	query, args, err := r.sb.Insert("trip_status_history").
		Columns("trip_id", "status", "changed_at").
		Values(record.TripID, record.Status, record.ChangedAt).
		ToSql()
	if err != nil {
		return err
	}

	exec := transaction.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	return err
}
