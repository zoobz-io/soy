package soy

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/sentinel"
)

type waitPolicyJob struct {
	ID        int    `db:"id" type:"integer" constraints:"primarykey"`
	Status    string `db:"status" type:"text"`
	CreatedAt string `db:"created_at" type:"timestamptz"`
}

func newWaitPolicySoy(t *testing.T) *Soy[waitPolicyJob] {
	t.Helper()
	sentinel.Tag("db")
	sentinel.Tag("type")
	sentinel.Tag("constraints")

	db := &sqlx.DB{}
	s, err := New[waitPolicyJob](db, "jobs", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return s
}

func TestSelect_WaitPolicy(t *testing.T) {
	s := newWaitPolicySoy(t)

	t.Run("FOR UPDATE SKIP LOCKED", func(t *testing.T) {
		result, err := s.Select().
			Where("id", "=", "job_id").
			ForUpdate().
			SkipLocked().
			Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, "FOR UPDATE SKIP LOCKED") {
			t.Errorf("SQL missing FOR UPDATE SKIP LOCKED: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("FOR UPDATE NOWAIT", func(t *testing.T) {
		result, err := s.Select().
			Where("id", "=", "job_id").
			ForUpdate().
			NoWait().
			Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, "FOR UPDATE NOWAIT") {
			t.Errorf("SQL missing FOR UPDATE NOWAIT: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("FOR SHARE SKIP LOCKED", func(t *testing.T) {
		result, err := s.Select().
			ForShare().
			SkipLocked().
			Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, "FOR SHARE SKIP LOCKED") {
			t.Errorf("SQL missing FOR SHARE SKIP LOCKED: %s", result.SQL)
		}
	})

	t.Run("SKIP LOCKED without lock mode errors at render", func(t *testing.T) {
		_, err := s.Select().
			Where("id", "=", "job_id").
			SkipLocked().
			Render()
		if err == nil {
			t.Fatal("expected error when SKIP LOCKED is used without a lock mode")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestQuery_WaitPolicy(t *testing.T) {
	s := newWaitPolicySoy(t)

	t.Run("FOR UPDATE SKIP LOCKED", func(t *testing.T) {
		result, err := s.Query().
			Where("status", "=", "pending").
			OrderBy("created_at", "asc").
			LimitParam("batch").
			ForUpdate().
			SkipLocked().
			Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, "FOR UPDATE SKIP LOCKED") {
			t.Errorf("SQL missing FOR UPDATE SKIP LOCKED: %s", result.SQL)
		}
		t.Logf("SQL: %s", result.SQL)
	})

	t.Run("FOR UPDATE NOWAIT", func(t *testing.T) {
		result, err := s.Query().
			ForUpdate().
			NoWait().
			Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if !strings.Contains(result.SQL, "FOR UPDATE NOWAIT") {
			t.Errorf("SQL missing FOR UPDATE NOWAIT: %s", result.SQL)
		}
	})

	t.Run("NOWAIT without lock mode errors at render", func(t *testing.T) {
		_, err := s.Query().
			NoWait().
			Render()
		if err == nil {
			t.Fatal("expected error when NOWAIT is used without a lock mode")
		}
		t.Logf("got expected error: %v", err)
	})
}
