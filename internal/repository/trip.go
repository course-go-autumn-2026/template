package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5"
)

// Определяем кастомные ошибки для ручек
var (
	ErrDriverBusy    = errors.New("driver is busy with another active trip")
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
)

type TripRepository struct {
	txManager *TxManager
}

func NewTripRepository(txManager *TxManager) *TripRepository {
	return &TripRepository{txManager: txManager}
}

// CreateTrip атомарно создает поездку
func (r *TripRepository) CreateTrip(ctx context.Context, t *Trip) error {
	db := r.txManager.GetQueryer(ctx)

	// 1. Вставляем саму поездку в таблицу trips
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trips").
		Columns("id", "user_id", "driver_id", "price", "status").
		Values(t.ID, t.UserID, t.DriverID, t.Price, t.Status).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip sql: %w", err)
	}

	_, err = db.Exec(ctx, sql, args...)
	if err != nil {
		// Проверяем код ошибки Postgres 23505 (unique_violation)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDriverBusy
		}
		return fmt.Errorf("execute insert trip: %w", err)
	}

	// 2. Вставляем стартовый статус в журнал истории
	sqlHist, argsHist, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "status").
		Values(t.ID, t.Status).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history sql: %w", err)
	}

	_, err = db.Exec(ctx, sqlHist, argsHist...)
	if err != nil {
		return fmt.Errorf("execute insert history: %w", err)
	}

	return nil
}

// GetTrip возвращает поездку целиком по её ID
func (r *TripRepository) GetTrip(ctx context.Context, id string) (*Trip, error) {
	db := r.txManager.GetQueryer(ctx)

	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "user_id", "driver_id", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip sql: %w", err)
	}

	var t Trip
	row := db.QueryRow(ctx, sql, args...)
	err = row.Scan(&t.ID, &t.UserID, &t.DriverID, &t.Price, &t.Status, &t.StartedAt, &t.FinishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTripNotFound // Поездки нет -> 404
		}
		return nil, fmt.Errorf("scan trip row: %w", err)
	}

	return &t, nil
}

// FinishTrip безопасно переводит поездку в completed с защитой от гонки
func (r *TripRepository) FinishTrip(ctx context.Context, id string) error {
	db := r.txManager.GetQueryer(ctx)

	// Блокируем строку поездки для текущей транзакции через FOR UPDATE
	selectSql, selectArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("status").
		From("trips").
		Where(sq.Eq{"id": id}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build select for update sql: %w", err)
	}

	var status string
	err = db.QueryRow(ctx, selectSql, selectArgs...).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTripNotFound
		}
		return fmt.Errorf("execute select for update: %w", err)
	}

	if status == "completed" {
		return ErrTripCompleted
	}

	// Обновляем статус
	updateSql, updateArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("trips").
		Set("status", "completed").
		Set("finished_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip sql: %w", err)
	}

	_, err = db.Exec(ctx, updateSql, updateArgs...)
	if err != nil {
		return fmt.Errorf("execute update trip: %w", err)
	}

	// Записываем изменение в историю статусов
	histSql, histArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "status").
		Values(id, "completed").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history finish sql: %w", err)
	}

	_, err = db.Exec(ctx, histSql, histArgs...)
	if err != nil {
		return fmt.Errorf("execute insert history finish: %w", err)
	}

	return nil
}
