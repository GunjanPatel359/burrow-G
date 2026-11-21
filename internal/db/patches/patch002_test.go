package patches

import (
	"context"
	"testing"
)

func TestPatch002_DropsBookmarkedColumn(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1)`)

	if err := Run(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	var cols string
	if err := conn.QueryRow(`SELECT group_concat(name) FROM pragma_table_info('sessions')`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != "agent,id,title,dir,last_used" {
		t.Errorf("columns = %s, want bookmarked gone", cols)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("rows = %d, want 1 kept", n)
	}
	if version, err := GetPatchVersion(context.Background(), conn); err != nil || version != "0.0.2" {
		t.Errorf("version = %q, %v; want 0.0.2", version, err)
	}
}

func TestPatch002_LeavesBookmarksUnchanged(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1),
			('claude', 'plain', 'Plain', '/tmp', 1, 0),
			('codex', 'kept', 'Kept', '/tmp', 1, 1);
		CREATE TABLE bookmarks (agent TEXT NOT NULL, id TEXT NOT NULL, PRIMARY KEY (agent, id));
		INSERT INTO bookmarks VALUES ('claude', 'kept'), ('claude', 'saved')`)

	if err := Run(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	var bookmarks string
	if err := conn.QueryRow(`SELECT group_concat(agent || '/' || id)
		FROM (SELECT agent, id FROM bookmarks ORDER BY agent, id)`).Scan(&bookmarks); err != nil {
		t.Fatal(err)
	}
	if bookmarks != "claude/kept,claude/saved" {
		t.Errorf("bookmarks = %q, want claude/kept,claude/saved", bookmarks)
	}
}

func TestPatch002_AlreadyAppliedIsSkipped(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1);
		CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO meta VALUES ('version', '0.0.2')`)

	for range 2 {
		if err := Run(context.Background(), conn); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("sessions = %d, want 1", n)
	}
	if version, err := GetPatchVersion(context.Background(), conn); err != nil || version != "0.0.2" {
		t.Errorf("version = %q, %v; want 0.0.2", version, err)
	}
}

func TestPatch002_VersionFailureRollsBackMigration(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1);
		CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL CHECK (value = '0.0.1'));
		INSERT INTO meta VALUES ('version', '0.0.1')`)

	if err := Run(context.Background(), conn); err == nil {
		t.Fatal("expected the version update to fail")
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'bookmarked'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("the legacy column should remain after rollback")
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE bookmarked = 1`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("the legacy bookmark flag should remain after rollback")
	}
	if version, err := GetPatchVersion(context.Background(), conn); err != nil || version != "0.0.1" {
		t.Errorf("version = %q, %v; want 0.0.1", version, err)
	}
}
