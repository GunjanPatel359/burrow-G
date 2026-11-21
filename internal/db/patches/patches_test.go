package patches

import (
	"context"
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
	conn.SetMaxOpenConns(1)
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
	invoke  func(context.Context, PatchContext) error
}

func (f fake) Version() string { return f.version }

func (f fake) Invoke(ctx context.Context, pct PatchContext) error {
	if f.invoke == nil {
		return nil
	}
	return f.invoke(ctx, pct)
}

// recorder returns a chain of empty patches that note when they run.
func recorder(ran *[]string, versions ...string) []Patch {
	var chain []Patch
	for _, v := range versions {
		chain = append(chain, fake{v, func(context.Context, PatchContext) error { *ran = append(*ran, v); return nil }})
	}
	return chain
}

func useChain(t *testing.T, patches []Patch) {
	t.Helper()
	previous := chain
	chain = patches
	t.Cleanup(func() { chain = previous })
}

func TestVersionCheck(t *testing.T) {
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
		if got := VersionCheck(tt.a, tt.b); got != tt.want {
			t.Errorf("VersionCheck(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestChain_InReleaseOrder(t *testing.T) {
	for i := 1; i < len(chain); i++ {
		if !VersionCheck(chain[i].Version(), chain[i-1].Version()) {
			t.Errorf("patch %s comes after %s", chain[i].Version(), chain[i-1].Version())
		}
	}
}

func TestGetPatchVersion(t *testing.T) {
	ctx := context.Background()
	conn := rawDB(t)
	if v, err := GetPatchVersion(ctx, conn); err != nil || v != "0.0.1" {
		t.Errorf("missing table: version = %q, %v; want 0.0.1", v, err)
	}

	mustExec(t, conn, `CREATE TABLE sessions (id TEXT)`)
	if v, err := GetPatchVersion(ctx, conn); err != nil || v != "0.0.1" {
		t.Errorf("file with no version: version = %q, %v; want 0.0.1", v, err)
	}

	mustExec(t, conn, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
	if v, err := GetPatchVersion(ctx, conn); err != nil || v != "0.0.1" {
		t.Errorf("missing row: version = %q, %v; want 0.0.1", v, err)
	}
	mustExec(t, conn, `INSERT INTO meta VALUES ('version', '0.0.2')`)
	if v, err := GetPatchVersion(ctx, conn); err != nil || v != "0.0.2" {
		t.Errorf("version = %q, %v; want 0.0.2", v, err)
	}

	mustExec(t, conn, `UPDATE meta SET value = '0.0.9'`)
	if v, err := GetPatchVersion(ctx, conn); err != nil || v != "0.0.9" {
		t.Errorf("stored newer version = %q, %v; want 0.0.9", v, err)
	}
}

func TestLatestVersion(t *testing.T) {
	t.Run("no patches", func(t *testing.T) {
		useChain(t, nil)
		if v := LatestVersion(); v != "0.0.1" {
			t.Errorf("latest = %s, want 0.0.1", v)
		}
	})
	t.Run("last registered patch", func(t *testing.T) {
		useChain(t, []Patch{fake{version: "0.0.2"}, fake{version: "0.0.10"}})
		if v := LatestVersion(); v != "0.0.10" {
			t.Errorf("latest = %s, want 0.0.10", v)
		}
	})
}

func TestRun_OnlyRunsIfNeeded(t *testing.T) {
	for _, version := range []string{"0.0.1", "0.0.2"} {
		t.Run(version, func(t *testing.T) {
			conn := rawDB(t)
			mustExec(t, conn, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
			if _, err := conn.Exec(`INSERT INTO meta VALUES ('version', ?)`, version); err != nil {
				t.Fatal(err)
			}
			ran := false
			useChain(t, []Patch{fake{"0.0.2", func(context.Context, PatchContext) error {
				ran = true
				return nil
			}}})
			if err := Run(context.Background(), conn); err != nil {
				t.Fatal(err)
			}
			if want := version == "0.0.1"; ran != want {
				t.Errorf("ran = %v, want %v", ran, want)
			}
		})
	}
}

func TestRun_DefaultVersionRunsNewerPatches(t *testing.T) {
	conn := rawDB(t)
	var ran []string
	useChain(t, recorder(&ran, "0.0.1", "0.0.2", "0.0.3"))

	if err := Run(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	if strings.Join(ran, ",") != "0.0.2,0.0.3" {
		t.Errorf("ran %v, want 0.0.2 then 0.0.3", ran)
	}
	if v := stored(t, conn); v != "0.0.3" {
		t.Errorf("version = %s, want 0.0.3", v)
	}
}

func TestRun_OnlyNewerPatchesInOrder(t *testing.T) {
	conn := rawDB(t)
	var ran []string
	useChain(t, recorder(&ran, "0.0.2", "0.0.3", "0.0.4"))
	mustExec(t, conn, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO meta VALUES ('version', '0.0.2')`)

	if err := Run(context.Background(), conn); err != nil {
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
	useChain(t, []Patch{
		fake{"0.0.2", func(ctx context.Context, pct PatchContext) error {
			_, err := pct.Tx.ExecContext(ctx, `CREATE TABLE a (x)`)
			return err
		}},
		fake{"0.0.3", func(ctx context.Context, pct PatchContext) error {
			if _, err := pct.Tx.ExecContext(ctx, `CREATE TABLE b (x)`); err != nil {
				return err
			}
			return errors.New("boom")
		}},
	})

	err := Run(context.Background(), conn)
	if err == nil || !strings.Contains(err.Error(), "0.0.3") {
		t.Errorf("err = %v, want it to name patch 0.0.3", err)
	}
	if v := stored(t, conn); v != "0.0.2" {
		t.Errorf("version = %s, want 0.0.2", v)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'b'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("table b from the failed patch should have been rolled back")
	}
}
