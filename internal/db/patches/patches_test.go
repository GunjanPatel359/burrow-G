package patches

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// rawDB opens an empty SQLite file.
func rawDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func userVersion(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var v int
	if err := conn.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRun_EachPatchOnceInOrder(t *testing.T) {
	conn := rawDB(t)
	var ran []string
	chain := []Patch{
		{"0.0.2", func(*sql.Tx) error { ran = append(ran, "0.0.2"); return nil }},
		{"0.0.3", func(*sql.Tx) error { ran = append(ran, "0.0.3"); return nil }},
	}

	for range 2 {
		if err := run(conn, chain); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(ran, ",") != "0.0.2,0.0.3" {
		t.Errorf("ran %v, want each patch once, in order", ran)
	}
	if v := userVersion(t, conn); v != 2 {
		t.Errorf("user_version = %d, want 2", v)
	}
}

func TestRun_FailedPatchChangesNothing(t *testing.T) {
	conn := rawDB(t)
	chain := []Patch{
		{"0.0.2", func(tx *sql.Tx) error { _, err := tx.Exec(`CREATE TABLE a (x)`); return err }},
		{"0.0.3", func(tx *sql.Tx) error {
			if _, err := tx.Exec(`CREATE TABLE b (x)`); err != nil {
				return err
			}
			return errors.New("boom")
		}},
	}

	err := run(conn, chain)
	if err == nil || !strings.Contains(err.Error(), "0.0.3") {
		t.Errorf("err = %v, want it to name patch 0.0.3", err)
	}
	if v := userVersion(t, conn); v != 1 {
		t.Errorf("user_version = %d, want 1 (only 0.0.2)", v)
	}
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'b'`).Scan(&n)
	if n != 0 {
		t.Error("table b from the failed patch should have been rolled back")
	}
}

func TestRun_RefusesNewerIndex(t *testing.T) {
	conn := rawDB(t)
	conn.Exec(`PRAGMA user_version = 5`)

	err := run(conn, []Patch{{"0.0.2", func(*sql.Tx) error { return nil }}})
	if err == nil || !strings.Contains(err.Error(), "newer burrow-G") {
		t.Errorf("err = %v, want a 'newer burrow-G' error", err)
	}
}

func TestPending(t *testing.T) {
	conn := rawDB(t)
	chain := []Patch{{"0.0.2", func(*sql.Tx) error { return nil }}}

	if p, err := pending(conn, chain); err != nil || !p {
		t.Errorf("new file: pending = %v, %v; want true", p, err)
	}
	run(conn, chain)
	if p, err := pending(conn, chain); err != nil || p {
		t.Errorf("after run: pending = %v, %v; want false", p, err)
	}
	conn.Exec(`PRAGMA user_version = 5`)
	if _, err := pending(conn, chain); err == nil || !strings.Contains(err.Error(), "newer burrow-G") {
		t.Errorf("newer file: err = %v, want a 'newer burrow-G' error", err)
	}
}
