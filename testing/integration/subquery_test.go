package integration

import (
	"context"
	"testing"

	"github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/soy"
)

// seedUsers inserts n users with ascending ages (10, 20, 30, ...) and returns
// the soy instance for test_users.
func seedUsers(t *testing.T, n int) (*soy.Soy[TestUser], context.Context) {
	t.Helper()
	db := getTestDB(t)
	truncateTestTable(t, db)

	c, err := soy.New[TestUser](db, "test_users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	for i := 1; i <= n; i++ {
		email := "user" + string(rune('a'+i-1)) + "@example.com"
		if _, err := c.Insert().Exec(ctx, &TestUser{
			Email: email,
			Name:  "pending",
			Age:   intPtr(i * 10),
		}); err != nil {
			t.Fatalf("seed insert %d failed: %v", i, err)
		}
	}
	return c, ctx
}

// TestSubqueryClaim_Integration exercises the full work-queue claim against a real
// PostgreSQL instance: UPDATE ... WHERE id IN (SELECT ... FOR UPDATE SKIP LOCKED)
// RETURNING, returning the whole claimed batch via ExecMany.
func TestSubqueryClaim_Integration(t *testing.T) {
	c, ctx := seedUsers(t, 5)

	// Claim the two youngest "pending" rows.
	pending := c.Query().Fields("id").
		Where("name", "=", "status_pending").
		OrderBy("age", "asc").
		LimitParam("batch").
		ForUpdate().
		SkipLocked()

	claimed, err := c.Modify().
		Set("name", "status_claimed").
		WhereInSubquery("id", pending).
		ExecMany(ctx, map[string]any{
			"status_claimed":     "claimed",
			"sq1_status_pending": "pending",
			"sq1_batch":          2,
		})
	if err != nil {
		t.Fatalf("ExecMany() failed: %v", err)
	}

	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed rows, got %d", len(claimed))
	}
	for _, u := range claimed {
		if u.Name != "claimed" {
			t.Errorf("claimed row %d has name %q, want claimed", u.ID, u.Name)
		}
	}

	// Exactly two rows should now be claimed; the other three stay pending.
	remaining, err := c.Count().
		Where("name", "=", "still_pending").
		Exec(ctx, map[string]any{"still_pending": "pending"})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if remaining != 3 {
		t.Errorf("expected 3 pending rows remaining, got %.0f", remaining)
	}
}

// TestSubqueryConditions_Integration checks IN / NOT IN / EXISTS read paths.
func TestSubqueryConditions_Integration(t *testing.T) {
	c, ctx := seedUsers(t, 4)

	t.Run("WhereInSubquery", func(t *testing.T) {
		sub := c.Query().Fields("id").Where("age", "<=", "cutoff")
		rows, err := c.Query().
			WhereInSubquery("id", sub).
			Exec(ctx, map[string]any{"sq1_cutoff": 20})
		if err != nil {
			t.Fatalf("Exec() failed: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("expected 2 rows (age 10, 20), got %d", len(rows))
		}
	})

	t.Run("WhereNotInSubquery", func(t *testing.T) {
		sub := c.Query().Fields("id").Where("age", "<=", "cutoff")
		rows, err := c.Query().
			WhereNotInSubquery("id", sub).
			Exec(ctx, map[string]any{"sq1_cutoff": 20})
		if err != nil {
			t.Fatalf("Exec() failed: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("expected 2 rows (age 30, 40), got %d", len(rows))
		}
	})
}
