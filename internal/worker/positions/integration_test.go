//go:build integration

package positions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestScanBatchesAndQueueLimit(t *testing.T) {
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
	ids := make([]uuid.UUID, 250)
	at := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	for i := range ids {
		ids[i] = uuid.New()
		_, err = db.Exec(ctx, `INSERT INTO trips(id,user_id,driver_id,start_latitude,start_longitude,end_latitude,end_longitude,price,status,started_at) VALUES($1,$2,$3,0,0,0,0,1,'active',$4)`, ids[i], uuid.New(), uuid.New(), at)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() { _, _ = db.Exec(ctx, `DELETE FROM trips WHERE id=ANY($1)`, ids) }()
	fresh, completed := uuid.New(), uuid.New()
	_, err = db.Exec(ctx, `INSERT INTO trips(id,user_id,driver_id,start_latitude,start_longitude,end_latitude,end_longitude,price,status,started_at,last_position_at) VALUES($1,$2,$3,0,0,0,0,1,'active',now()-interval '2 hours',now())`, fresh, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(ctx, `INSERT INTO trips(id,user_id,driver_id,start_latitude,start_longitude,end_latitude,end_longitude,price,status,started_at,finished_at) VALUES($1,$2,$3,0,0,0,0,1,'completed',now()-interval '2 hours',now())`, completed, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = db.Exec(ctx, `DELETE FROM trips WHERE id=ANY($1)`, []uuid.UUID{fresh, completed}) }()

	p := makePool(t, 250, fakeGateway{make(chan uuid.UUID, 1)})
	p.db = db
	p.scan(ctx)
	if len(p.queue) != 250 {
		t.Fatalf("expected 250 tasks, got %d", len(p.queue))
	}
	seen := map[uuid.UUID]bool{}
	for _, task := range p.queue {
		if seen[task.ID] {
			t.Fatal("duplicate across batch boundary")
		}
		seen[task.ID] = true
	}
	limited := makePool(t, 50, fakeGateway{make(chan uuid.UUID, 1)})
	limited.db = db
	limited.scan(ctx)
	if len(limited.queue) != 50 {
		t.Fatalf("expected 50 tasks, got %d", len(limited.queue))
	}
}
