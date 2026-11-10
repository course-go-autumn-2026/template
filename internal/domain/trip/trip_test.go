package trip

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTripFinish(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		status  Status
		wantErr error
	}{
		{name: "active trip finishes", status: StatusActive, wantErr: nil},
		{name: "completed trip cannot finish", status: StatusCompleted, wantErr: ErrTripCompleted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := Trip{Id: uuid.New(), Status: tc.status}
			err := tr.Finish(now)
			if err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if tr.Status != StatusCompleted {
					t.Fatalf("status = %v, want completed", tr.Status)
				}
				if tr.FinishedAt == nil {
					t.Fatal("FinishedAt is nil")
				}
			}
		})
	}
}
