package postgres

import "testing"

func TestCollectionName(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{"SELECT id FROM trips WHERE id=$1", "trips"},
		{"INSERT INTO trip_status_history (trip_id) VALUES ($1)", "trip_status_history"},
		{"UPDATE trip_idempotency_keys SET trip_id=$1", "trip_idempotency_keys"},
		{"SELECT 1", "unknown"},
	} {
		if got := collectionName(tc.sql); got != tc.want {
			t.Errorf("collectionName(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}
