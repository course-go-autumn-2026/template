package idempotency

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/George1487/MalyginGeorge/internal/infra/postgres/transaction"
)

type Record struct {
	RequestHash  []byte
	ResponseBody []byte
	ContentType  string
	Location     string
}

type databaseExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) LockAndGet(ctx context.Context, key uuid.UUID) (*Record, error) {
	executor := s.executor(ctx)

	lockQuery, lockArgs, err := sq.Select().
		Column("pg_advisory_xact_lock(?)", advisoryLockID(key)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build idempotency lock query: %w", err)
	}
	if _, err = executor.Exec(ctx, lockQuery, lockArgs...); err != nil {
		return nil, fmt.Errorf("lock idempotency key: %w", err)
	}

	deleteQuery, deleteArgs, err := sq.Delete("idempotency_keys").
		Where(sq.Eq{"idempotency_key": key}).
		Where(sq.LtOrEq{"expires_at": time.Now().UTC()}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build expired idempotency key query: %w", err)
	}
	if _, err = executor.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return nil, fmt.Errorf("delete expired idempotency key: %w", err)
	}

	selectQuery, selectArgs, err := sq.Select(
		"request_hash",
		"response_body",
		"content_type",
		"location",
	).
		From("idempotency_keys").
		Where(sq.Eq{"idempotency_key": key}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build idempotency key query: %w", err)
	}

	record := &Record{}
	err = executor.QueryRow(ctx, selectQuery, selectArgs...).Scan(
		&record.RequestHash,
		&record.ResponseBody,
		&record.ContentType,
		&record.Location,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}

	return record, nil
}

func (s *Store) Save(
	ctx context.Context,
	key uuid.UUID,
	requestHash []byte,
	responseBody []byte,
	contentType string,
	location string,
	expiresAt time.Time,
) error {
	query, args, err := sq.Insert("idempotency_keys").
		Columns(
			"idempotency_key",
			"request_hash",
			"response_body",
			"content_type",
			"location",
			"expires_at",
		).
		Values(
			key,
			requestHash,
			responseBody,
			contentType,
			location,
			expiresAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save idempotency key query: %w", err)
	}

	if _, err = s.executor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}

	return nil
}

func (s *Store) executor(ctx context.Context) databaseExecutor {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}

	return s.pool
}

func advisoryLockID(key uuid.UUID) int64 {
	// #nosec G115 -- casting 8-byte UUID prefix to int64 for postgres advisory lock key
	return int64(binary.BigEndian.Uint64(key[:8]))
}
