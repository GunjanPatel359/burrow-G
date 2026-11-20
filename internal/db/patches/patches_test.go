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

func mustExec(t *testing.T, conn *sql.DB, query string) {
	t.Helper()
	if _, err := conn.Exec(query); err != nil {
		t.Fatal(err)
	}
}

func stored(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var v string
	if err := conn.QueryRow(`SELECT value FROM meta WHERE key = 'version'`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

type fake struct {
	version string
	invoke  func() error
}

func (f fake) Version() string { return f.version }

func (f fake) Invoke() error {
	if f.invoke == nil {
		return nil
	}
	return f.invoke()
}

// recorder returns a chain of empty patches that note when they run.
func recorder(ran *[]string, versions ...string) []Patch {
	var chain []Patch
	for _, v := range versions {
		chain = append(chain, fake{v, func() error { *ran = append(*ran, v); return nil }})
	}
	return chain
}

func TestNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"0.0.2", "0.0.1", true},
		{"0.0.1", "0.0.2", false},
		{"0.0.2", "0.0.2", false},
		{"0.0.10", "0.0.9", true},
		{"0.1.0", "0.0.9", true},
		{"1.0.0", "0.9.9", true},
	}
	for _, tt := range tests {
		if got := newer(tt.a, tt.b); got != tt.want {
			t.Errorf("newer(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestChain_InReleaseOrder(t *testing.T) {
	for i := 1; i < len(chain); i++ {
		if !newer(chain[i].Version(), chain[i-1].Version()) {
			t.Errorf("patch %s comes after %s", chain[i].Version(), chain[i-1].Version())
		}
	}
}

func TestFrom(t *testing.T) {
	chain := []Patch{fake{version: "0.0.2"}}

	conn := rawDB(t)
	if v, err := from(conn, chain); err != nil || v != "" {
		t.Errorf("new file: from = %q, %v; want \"\"", v, err)
	}

	mustExec(t, conn, `CREATE TABLE sessions (id TEXT)`)
	if v, err := from(conn, chain); err != nil || v != "0.0.1" {
		t.Errorf("file with no version: from = %q, %v; want 0.0.1", v, err)
	}

	mustExec(t, conn, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO meta VALUES ('version', '0.0.2')`)
	if v, err := from(conn, chain); err != nil || v != "0.0.2" {
		t.Errorf("from = %q, %v; want 0.0.2", v, err)
	}

	mustExec(t, conn, `UPDATE meta SET value = '0.0.9'`)
	if _, err := from(conn, chain); err == nil || !strings.Contains(err.Error(), "newer burrow-G") {
		t.Errorf("newer file: err = %v, want a 'newer burrow-G' error", err)
	}
}

func TestNeeded(t *testing.T) {
	chain := []Patch{fake{version: "0.0.2"}}
	for v, want := range map[string]bool{"": false, "0.0.1": true, "0.0.2": false} {
		if got := needed(chain, v); got != want {
			t.Errorf("needed(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestRun_NewFileOnlyGetsVersion(t *testing.T) {
	conn := rawDB(t)
	var ran []string

	if err := run(conn, recorder(&ran, "0.0.2", "0.0.3"), ""); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 {
		t.Errorf("ran %v on a new file, want none", ran)
	}
	if v := stored(t, conn); v != "0.0.3" {
		t.Errorf("version = %s, want 0.0.3", v)
	}
}

func TestRun_OnlyNewerPatchesInOrder(t *testing.T) {
	conn := rawDB(t)
	var ran []string
	chain := recorder(&ran, "0.0.2", "0.0.3", "0.0.4")

	if err := run(conn, chain, "0.0.2"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(ran, ",") != "0.0.3,0.0.4" {
		t.Errorf("ran %v, want 0.0.3 then 0.0.4", ran)
	}
	if v := stored(t, conn); v != "0.0.4" {
		t.Errorf("version = %s, want 0.0.4", v)
	}
}

func TestRun_FailedPatchChangesNothing(t *testing.T) {
	conn := rawDB(t)
	chain := []Patch{
		fake{"0.0.2", func() error { return exec(`CREATE TABLE a (x)`) }},
		fake{"0.0.3", func() error {
			if err := exec(`CREATE TABLE b (x)`); err != nil {
				return err
			}
			return errors.New("boom")
		}},
	}

	err := run(conn, chain, "0.0.1")
	if err == nil || !strings.Contains(err.Error(), "0.0.3") {
		t.Errorf("err = %v, want it to name patch 0.0.3", err)
	}
	if v := stored(t, conn); v != "0.0.2" {
		t.Errorf("version = %s, want 0.0.2", v)
	}
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'b'`).Scan(&n)
	if n != 0 {
		t.Error("table b from the failed patch should have been rolled back")
	}
}
