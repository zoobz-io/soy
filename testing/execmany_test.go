package testing

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/soy"
)

// execManyJob mirrors a work-queue row for end-to-end ExecMany tests.
type execManyJob struct {
	ID     int    `db:"id" type:"integer" constraints:"primarykey"`
	Status string `db:"status" type:"varchar(255)"`
}

// TestExecMany_OutboxClaim drives the full batch claim through the mock driver:
// UPDATE ... WHERE id IN (SELECT ... FOR UPDATE SKIP LOCKED) RETURNING ...,
// returning every claimed row rather than erroring on multiple rows.
func TestExecMany_OutboxClaim(t *testing.T) {
	mock := NewMockDB(t)
	mock.ExpectQuery().WithRows([]execManyJob{
		{ID: 1, Status: "processing"},
		{ID: 2, Status: "processing"},
		{ID: 3, Status: "processing"},
	})

	jobs, err := soy.New[execManyJob](mock.DB(), "jobs", postgres.New())
	if err != nil {
		t.Fatalf("soy.New() error: %v", err)
	}

	pending := jobs.Query().Fields("id").
		Where("status", "=", "pending_status").
		LimitParam("batch").
		ForUpdate().
		SkipLocked()

	claimed, err := jobs.Modify().
		Set("status", "processing_status").
		WhereInSubquery("id", pending).
		ExecMany(context.Background(), map[string]any{
			"processing_status":  "processing",
			"sq1_pending_status": "pending",
			"sq1_batch":          3,
		})
	if err != nil {
		t.Fatalf("ExecMany() error: %v", err)
	}

	if len(claimed) != 3 {
		t.Fatalf("expected 3 claimed rows, got %d", len(claimed))
	}
	for i, want := range []int{1, 2, 3} {
		if claimed[i].ID != want {
			t.Errorf("claimed[%d].ID = %d, want %d", i, claimed[i].ID, want)
		}
		if claimed[i].Status != "processing" {
			t.Errorf("claimed[%d].Status = %q, want processing", i, claimed[i].Status)
		}
	}

	// The rendered UPDATE the driver saw must carry the subquery and wait policy.
	calls := mock.Calls()
	if len(calls) == 0 {
		t.Fatal("expected at least one recorded call")
	}
	sql := calls[0].Query
	for _, want := range []string{`UPDATE "jobs"`, `IN (SELECT`, "FOR UPDATE SKIP LOCKED", "RETURNING"} {
		if !strings.Contains(sql, want) {
			t.Errorf("recorded SQL missing %q: %s", want, sql)
		}
	}

	mock.AssertExpectations()
}

// TestExecMany_MultipleRows confirms ExecMany returns every updated row where the
// single-row Exec would error with "expected exactly one row updated, found multiple".
func TestExecMany_MultipleRows(t *testing.T) {
	mock := NewMockDB(t)
	mock.ExpectQuery().WithRows([]execManyJob{
		{ID: 10, Status: "done"},
		{ID: 11, Status: "done"},
	})

	jobs, err := soy.New[execManyJob](mock.DB(), "jobs", postgres.New())
	if err != nil {
		t.Fatalf("soy.New() error: %v", err)
	}

	updated, err := jobs.Modify().
		Set("status", "new_status").
		Where("status", "=", "old_status").
		ExecMany(context.Background(), map[string]any{
			"new_status": "done",
			"old_status": "queued",
		})
	if err != nil {
		t.Fatalf("ExecMany() error: %v", err)
	}
	if len(updated) != 2 {
		t.Fatalf("expected 2 updated rows, got %d", len(updated))
	}

	mock.AssertExpectations()
}
