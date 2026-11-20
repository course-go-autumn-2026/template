//go:build integration

package positions

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres/transaction"
)

type noopCancel struct{}

func (noopCancel) Cancel(context.Context, uuid.UUID) {}
func TestSavePositionTransaction(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.New()
	_, err = db.Exec(ctx, `INSERT INTO trips(id,user_id,driver_id,start_latitude,start_longitude,end_latitude,end_longitude,price,status,started_at) VALUES($1,$2,$3,0,0,0,0,1,'active',now())`, id, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = db.Exec(ctx, `DELETE FROM trips WHERE id=$1`, id) }()
	h, err := New(db, transaction.New(db, time.Second*5), noopCancel{}, time.Second*5, noop.NewMeterProvider().Meter("test"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	send := func(at time.Time) int {
		body := fmt.Sprintf(`{"coordinates":{"latitude":1,"longitude":2},"recorded_at":%q}`, at.UTC().Format(time.RFC3339))
		w := httptest.NewRecorder()
		h.Create(w, httptest.NewRequest("POST", "/", strings.NewReader(body)), id)
		return w.Code
	}
	now := time.Now().UTC().Truncate(time.Second)
	if got := send(now); got != 201 {
		t.Fatalf("save: %d", got)
	}
	if got := send(now.Add(-time.Hour)); got != 201 {
		t.Fatalf("old position: %d", got)
	}
	var last time.Time
	var count int
	if err = db.QueryRow(ctx, `SELECT last_position_at,(SELECT count(*) FROM trip_positions WHERE trip_id=$1) FROM trips WHERE id=$1`, id).Scan(&last, &count); err != nil {
		t.Fatal(err)
	}
	if !last.Equal(now) || count != 2 {
		t.Fatalf("last=%v count=%d", last, count)
	}
	// Force the update to fail after the insert; both writes must roll back.
	_, err = db.Exec(ctx, `CREATE FUNCTION reject_position_update() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$; CREATE TRIGGER reject_position_update BEFORE UPDATE OF last_position_at ON trips FOR EACH ROW EXECUTE FUNCTION reject_position_update()`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.Exec(ctx, `DROP TRIGGER reject_position_update ON trips; DROP FUNCTION reject_position_update()`)
	}()
	if got := send(now); got != 500 {
		t.Fatalf("expected rollback error, got %d", got)
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM trip_positions WHERE trip_id=$1`, id).Scan(&count); err != nil || count != 2 {
		t.Fatalf("rollback count=%d err=%v", count, err)
	}
}
