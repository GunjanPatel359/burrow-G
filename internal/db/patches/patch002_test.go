package patches

import (
	"context"
	"database/sql"
	"testing"
)

const oldFile = `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
	dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
	INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1);
	CREATE TABLE bookmarks (agent TEXT NOT NULL, id TEXT NOT NULL, PRIMARY KEY (agent, id));
	INSERT INTO bookmarks VALUES ('claude', 'kept')`

func tables(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var names string
	err := conn.QueryRow(`SELECT COALESCE(group_concat(name), '') FROM
		(SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('sessions', 'bookmarks') ORDER BY name)`).Scan(&names)
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func TestPatch002_DropsOldTables(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, oldFile)

	if err := Run(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	if got := tables(t, conn); got != "" {
		t.Errorf("tables left = %q, want sessions and bookmarks gone", got)
	}
	if version, err := GetPatchVersion(context.Background(), conn); err != nil || version != "0.0.2" {
		t.Errorf("version = %q, %v; want 0.0.2", version, err)
	}
}

func TestPatch002_NewFile(t *testing.T) {
	conn := rawDB(t)
	if err := Run(context.Background(), conn); err != nil {
		t.Errorf("a new file has nothing to drop: %v", err)
	}
}

func TestPatch002_AlreadyAppliedIsSkipped(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, oldFile+`;
		CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO meta VALUES ('version', '0.0.2')`)

	if err := Run(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	if got := tables(t, conn); got != "bookmarks,sessions" {
		t.Errorf("tables = %q, want both kept", got)
	}
}

func TestPatch002_VersionFailureRollsBack(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, oldFile+`;
		CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL CHECK (value = '0.0.1'));
		INSERT INTO meta VALUES ('version', '0.0.1')`)

	if err := Run(context.Background(), conn); err == nil {
		t.Fatal("expected the version update to fail")
	}
	if got := tables(t, conn); got != "bookmarks,sessions" {
		t.Errorf("tables = %q, want both back after rollback", got)
	}
}
