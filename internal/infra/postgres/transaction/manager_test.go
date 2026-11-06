package transaction

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeTx struct {
	pgx.Tx
	rollbackErr        error
	rollbackCtxErr     error
	rollbacks, commits int
}

func (tx *fakeTx) Rollback(ctx context.Context) error {
	tx.rollbacks++
	tx.rollbackCtxErr = ctx.Err()
	return tx.rollbackErr
}

func (tx *fakeTx) Commit(context.Context) error { tx.commits++; return nil }

type fakePool struct {
	tx     *fakeTx
	begins int
}

func (p *fakePool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.begins++
	return p.tx, nil
}

func TestPanicRollback(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		logged bool
	}{
		{"failed", errors.New("rollback unavailable"), true},
		{"closed", pgx.ErrTxClosed, false},
		{"success", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			tx := &fakeTx{rollbackErr: tc.err}
			pool := &fakePool{tx: tx}
			m := NewManager(pool, pgx.ReadCommitted, time.Second, slog.New(slog.NewJSONHandler(&logs, nil)))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			original := errors.New("original panic")
			func() {
				defer func() {
					if got := recover(); got != original {
						t.Fatalf("panic=%v", got)
					}
				}()
				_ = m.Do(ctx, func(inner context.Context) error {
					return m.Do(inner, func(context.Context) error { cancel(); panic(original) })
				})
			}()
			if pool.begins != 1 || tx.rollbacks != 1 || tx.commits != 0 || tx.rollbackCtxErr != nil {
				t.Fatalf("begins=%d rollbacks=%d commits=%d rollback context=%v", pool.begins, tx.rollbacks, tx.commits, tx.rollbackCtxErr)
			}
			if got := strings.Contains(logs.String(), "rollback after panic failed"); got != tc.logged {
				t.Fatalf("logs=%s", logs.String())
			}
		})
	}
}

func TestOrdinaryRollbackError(t *testing.T) {
	var logs bytes.Buffer
	tx := &fakeTx{rollbackErr: errors.New("rollback unavailable")}
	m := NewManager(&fakePool{tx: tx}, pgx.ReadCommitted, time.Second, slog.New(slog.NewJSONHandler(&logs, nil)))
	original := errors.New("function failed")
	err := m.Do(context.Background(), func(context.Context) error { return original })
	if !errors.Is(err, original) || !strings.Contains(err.Error(), "rollback failed") || logs.Len() != 0 {
		t.Fatalf("err=%v logs=%s", err, logs.String())
	}
}
