package transaction

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
	begins   int
}

func (f *fakeBeginner) Begin(context.Context) (Tx, error) {
	f.begins++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.tx, nil
}

type fakeTx struct {
	commits   int
	rollbacks int
}

func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (f *fakeTx) Commit(context.Context) error {
	f.commits++
	return nil
}

func (f *fakeTx) Rollback(context.Context) error {
	f.rollbacks++
	return nil
}

func TestWithinTransactionCommit(t *testing.T) {
	tx := &fakeTx{}
	manager := NewManager(&fakeBeginner{tx: tx})

	err := manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatal("expected transaction in context")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestWithinTransactionRollbackOnError(t *testing.T) {
	tx := &fakeTx{}
	manager := NewManager(&fakeBeginner{tx: tx})
	wantErr := errors.New("operation failed")

	err := manager.WithinTransaction(context.Background(), func(context.Context) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

func TestNestedTransactionReusesExistingTransaction(t *testing.T) {
	tx := &fakeTx{}
	beginner := &fakeBeginner{tx: tx}
	manager := NewManager(beginner)

	err := manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		outerTx, _ := TxFromContext(ctx)
		return manager.WithinTransaction(ctx, func(nestedCtx context.Context) error {
			nestedTx, ok := TxFromContext(nestedCtx)
			if !ok || outerTx != nestedTx {
				t.Fatal("nested call should reuse the existing transaction")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if beginner.begins != 1 || tx.commits != 1 {
		t.Fatalf("begins=%d commits=%d", beginner.begins, tx.commits)
	}
}

func TestWithinTransactionRollbackOnPanic(t *testing.T) {
	tx := &fakeTx{}
	manager := NewManager(&fakeBeginner{tx: tx})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic to be propagated")
		}
		if tx.rollbacks != 1 {
			t.Fatalf("expected one rollback, got %d", tx.rollbacks)
		}
	}()

	_ = manager.WithinTransaction(context.Background(), func(context.Context) error {
		panic("unexpected failure")
	})
}
