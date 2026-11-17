//go:build integration

package trip_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

func TestStaleTrips(t *testing.T) {
	ctx, pool, repo, _ := database(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	stale, never, fresh, completed := newTrip(t), newTrip(t), newTrip(t), newTrip(t)

	for i, entity := range []*trip.Trip{&stale, &never, &fresh, &completed} {
		entity.StartedAt = now.Add(-time.Duration(10-i) * time.Minute)

		if err := repo.CreateTrip(ctx, *entity); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := pool.Exec(ctx, "UPDATE trips SET last_position_at=$2 WHERE id=$1", fresh.ID, now); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(
		ctx,
		"UPDATE trips SET last_position_at=$2 WHERE id=$1",
		stale.ID,
		now.Add(-5*time.Minute),
	); err != nil {
		t.Fatal(err)
	}

	if err := repo.FinishTrip(ctx, completed.ID, now); err != nil {
		t.Fatal(err)
	}

	actual, err := repo.FindStaleTrips(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if len(actual) != 2 || actual[0].ID != never.ID || actual[1].ID != stale.ID || actual[0].DriverID != never.DriverID {
		t.Fatalf("unexpected stale trips: %+v", actual)
	}

	for _, entity := range []trip.Trip{stale, never, fresh, completed, {ID: uuid.New()}} {
		needed, err := repo.NeedsPosition(ctx, entity.ID, time.Minute)
		want := entity.ID == stale.ID || entity.ID == never.ID

		if err != nil || needed != want {
			t.Fatalf("needed=%v want=%v: %v", needed, want, err)
		}
	}
}

func TestPositionAtomicityAndOrdering(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	entity := newTrip(t)

	if err := repo.CreateTrip(ctx, entity); err != nil {
		t.Fatal(err)
	}

	recent := time.Now().UTC().Truncate(time.Microsecond)

	var saved []trip.Position

	for _, at := range []time.Time{recent, recent.Add(-time.Minute), recent} {
		err := tx.Do(ctx, func(ctx context.Context) error {
			if _, err := repo.GetTripForUpdate(ctx, entity.ID); err != nil {
				return err
			}

			p, err := repo.SavePosition(ctx, entity.ID, trip.PositionData{RecordedAt: at})
			saved = append(saved, p)

			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	actual, err := repo.GetTrip(ctx, entity.ID)

	if err != nil || actual.LastPositionAt == nil || !actual.LastPositionAt.Equal(recent) {
		t.Fatalf("timestamp regressed: %+v: %v", actual, err)
	}

	route, err := repo.ListPositions(ctx, entity.ID)

	if err != nil || len(route) != 3 || route[0].ID != saved[1].ID || route[1].ID != saved[0].ID || route[2].ID != saved[2].ID {
		t.Fatalf("route order: %+v: %v", route, err)
	}

	_, err = pool.Exec(ctx, "ALTER TABLE trips ADD CONSTRAINT test_reject_position CHECK (last_position_at IS NULL OR last_position_at < now() + interval '1 hour')")
	if err != nil {
		t.Fatal(err)
	}

	err = tx.Do(ctx, func(ctx context.Context) error {
		_, err := repo.SavePosition(ctx, entity.ID, trip.PositionData{RecordedAt: recent.Add(2 * time.Hour)})

		return err
	})

	if err == nil {
		t.Fatal("constraint did not fail")
	}

	route, err = repo.ListPositions(ctx, entity.ID)

	if err != nil || len(route) != 3 {
		t.Fatalf("partial insert survived rollback: %+v: %v", route, err)
	}

	actual, err = repo.GetTrip(ctx, entity.ID)

	if err != nil || !actual.LastPositionAt.Equal(recent) {
		t.Fatal("rollback changed last position")
	}
}

func TestMigrationRoundTrip(t *testing.T) {
	ctx, pool, _, _ := database(t)
	files, err := filepath.Glob("../../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}

	slices.Reverse(files)

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		_, down, ok := strings.Cut(string(body), "-- +goose Down")

		if !ok {
			t.Fatalf("no down: %s", file)
		}

		if _, err := pool.Exec(ctx, down); err != nil {
			t.Fatalf("down %s: %v", file, err)
		}
	}

	slices.Reverse(files)

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		up, _, _ := strings.Cut(string(body), "-- +goose Down")

		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("up %s: %v", file, err)
		}
	}
}

func TestStaleQueryUsesIndex(t *testing.T) {
	ctx, pool, repo, _ := database(t)
	_, err := pool.Exec(ctx, `INSERT INTO trips (id,user_id,driver_id,start_latitude,start_longitude,end_latitude,end_longitude,price,status,started_at)
 SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),0,0,0,0,0,'active',now() FROM generate_series(1,20000)`)
	if err != nil {
		t.Fatal(err)
	}

	stale := newTrip(t)
	stale.StartedAt = time.Now().Add(-time.Hour)

	if err := repo.CreateTrip(ctx, stale); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, "ANALYZE trips"); err != nil {
		t.Fatal(err)
	}

	rows, err := pool.Query(ctx, `EXPLAIN (ANALYZE, BUFFERS) SELECT id,driver_id,COALESCE(last_position_at,started_at) FROM trips
 WHERE status='active' AND COALESCE(last_position_at,started_at) < now() - $1::double precision * interval '1 second'
 ORDER BY COALESCE(last_position_at,started_at),id`, float64(60))
	if err != nil {
		t.Fatal(err)
	}

	defer rows.Close()

	var plan strings.Builder

	for rows.Next() {
		var line string

		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}

		plan.WriteString(line)
		plan.WriteByte('\n')
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	t.Log(plan.String())

	if !strings.Contains(plan.String(), "trips_active_position_idx") || strings.Contains(plan.String(), "Seq Scan") {
		t.Fatal("stale query did not use its index")
	}
}
