package soy

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/sentinel"
)

type subqueryJob struct {
	ID        int    `db:"id" type:"integer" constraints:"primarykey"`
	Status    string `db:"status" type:"text"`
	CreatedAt string `db:"created_at" type:"timestamptz"`
}

type subqueryUser struct {
	ID     int    `db:"id" type:"integer" constraints:"primarykey"`
	Status string `db:"status" type:"text"`
}

func newSubquerySoys(t *testing.T) (*Soy[subqueryJob], *Soy[subqueryUser]) {
	t.Helper()
	sentinel.Tag("db")
	sentinel.Tag("type")
	sentinel.Tag("constraints")

	db := &sqlx.DB{}
	jobs, err := New[subqueryJob](db, "jobs", postgres.New())
	if err != nil {
		t.Fatalf("New[jobs]() failed: %v", err)
	}
	users, err := New[subqueryUser](db, "users", postgres.New())
	if err != nil {
		t.Fatalf("New[users]() failed: %v", err)
	}
	return jobs, users
}

// TestUpdate_OutboxClaim covers the motivating case from the issue: a concurrent
// work-queue claim built entirely through the query builder.
func TestUpdate_OutboxClaim(t *testing.T) {
	jobs, _ := newSubquerySoys(t)

	pending := jobs.Query().Fields("id").
		Where("status", "=", "pending_status").
		OrderBy("created_at", "asc").
		LimitParam("batch").
		ForUpdate().
		SkipLocked()

	upd := jobs.Modify().
		Set("status", "processing_status").
		WhereInSubquery("id", pending)

	if !upd.hasWhere {
		t.Fatal("WhereInSubquery must satisfy the mandatory-WHERE safety guard (hasWhere)")
	}

	result, err := upd.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	sql := result.SQL
	for _, want := range []string{
		`UPDATE "jobs"`,
		`IN (SELECT`,
		`FOR UPDATE SKIP LOCKED`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q: %s", want, sql)
		}
	}
	t.Logf("SQL: %s", sql)
}

func TestSubquery_AllBuilders(t *testing.T) {
	jobs, _ := newSubquerySoys(t)

	sub := func() *Query[subqueryJob] {
		return jobs.Query().Fields("id").Where("status", "=", "pending_status")
	}

	t.Run("Select WhereInSubquery", func(t *testing.T) {
		result, err := jobs.Select().WhereInSubquery("id", sub()).Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, `"id" IN (SELECT`) {
			t.Errorf("unexpected SQL: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("Query WhereNotInSubquery", func(t *testing.T) {
		result, err := jobs.Query().WhereNotInSubquery("id", sub()).Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, `"id" NOT IN (SELECT`) {
			t.Errorf("unexpected SQL: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("Query WhereExists", func(t *testing.T) {
		result, err := jobs.Query().WhereExists(sub()).Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, `EXISTS (SELECT`) {
			t.Errorf("unexpected SQL: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("Query WhereNotExists", func(t *testing.T) {
		result, err := jobs.Query().WhereNotExists(sub()).Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, `NOT EXISTS (SELECT`) {
			t.Errorf("unexpected SQL: %s", result.SQL)
		}
	})

	t.Run("Delete WhereInSubquery satisfies safety guard", func(t *testing.T) {
		del := jobs.Remove().WhereInSubquery("id", sub())
		if !del.hasWhere {
			t.Fatal("WhereInSubquery must satisfy the DELETE mandatory-WHERE safety guard")
		}
		result, err := del.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, `DELETE FROM "jobs"`) || !strings.Contains(result.SQL, `IN (SELECT`) {
			t.Errorf("unexpected SQL: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})
}

// TestSubquery_CrossTable verifies a subquery over a different table/row type.
func TestSubquery_CrossTable(t *testing.T) {
	jobs, users := newSubquerySoys(t)

	activeUsers := users.Query().Fields("id").Where("status", "=", "active_status")

	result, err := jobs.Query().WhereInSubquery("id", activeUsers).Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if !strings.Contains(result.SQL, `FROM "users"`) {
		t.Errorf("cross-table subquery should reference users: %s", result.SQL)
	}
	t.Logf("SQL: %s", result.SQL)
}

func TestSubquery_Errors(t *testing.T) {
	jobs, _ := newSubquerySoys(t)

	t.Run("nil subquery", func(t *testing.T) {
		_, err := jobs.Query().WhereInSubquery("id", nil).Render()
		if err == nil {
			t.Fatal("expected error for nil subquery")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("typed-nil subquery", func(t *testing.T) {
		var nilSub *Query[subqueryJob]
		_, err := jobs.Query().WhereInSubquery("id", nilSub).Render()
		if err == nil {
			t.Fatal("expected error for typed-nil subquery")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("subquery build error propagates without panic", func(t *testing.T) {
		badSub := jobs.Query().Where("nonexistent_field", "=", "x")
		_, err := jobs.Query().WhereInSubquery("id", badSub).Render()
		if err == nil {
			t.Fatal("expected error when subquery has a build error")
		}
		t.Logf("got expected error: %v", err)
	})

	t.Run("invalid outer field", func(t *testing.T) {
		sub := jobs.Query().Fields("id").Where("status", "=", "s")
		_, err := jobs.Query().WhereInSubquery("nonexistent_field", sub).Render()
		if err == nil {
			t.Fatal("expected error for invalid outer field")
		}
		t.Logf("got expected error: %v", err)
	})
}
