package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/db/patches"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var now = time.Date(2026, 11, 13, 12, 0, 0, 0, time.Local)

var (
	older = session.Session{Agent: "claude", ID: "old", Title: "Old one", Dir: "/tmp/a", LastUsed: now.Add(-24 * time.Hour)}
	newer = session.Session{Agent: "claude", ID: "new", Title: "New one", Dir: "/tmp/b", LastUsed: now.Add(-1 * time.Hour)}
)

func openTest(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "sub", "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func ids(t *testing.T, d *DB, agent string) []string {
	t.Helper()
	list, err := d.List(agent)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range list {
		out = append(out, s.ID)
	}
	return out
}

func TestSaveAndList(t *testing.T) {
	d := openTest(t)
	if err := d.Save("claude", []session.Session{older, newer}); err != nil {
		t.Fatal(err)
	}

	list, err := d.List("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != newer || list[1] != older {
		t.Errorf("got %+v\nwant [newer, older]", list)
	}

	counts, err := d.Counts()
	if err != nil {
		t.Fatal(err)
	}
	if counts["claude"] != 2 {
		t.Errorf("Counts()[claude] = %d, want 2", counts["claude"])
	}
}

func TestSetBookmark_OnAndOff(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})

	if err := d.SetBookmark(older, true); err != nil {
		t.Fatal(err)
	}
	if err := d.SetBookmark(older, true); err != nil {
		t.Errorf("bookmarking twice: %v", err)
	}
	list, _ := d.List("claude")
	if list[0].ID != "old" || !list[0].Bookmarked {
		t.Errorf("first = %+v, want old and bookmarked", list[0])
	}

	if err := d.SetBookmark(older, false); err != nil {
		t.Fatal(err)
	}
	list, _ = d.List("claude")
	if list[0].ID != "new" || list[1].Bookmarked {
		t.Errorf("got %+v, want new first and no bookmarks", list)
	}
}

func TestBookmark_HiddenWhileGoneBackWhenReturns(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.SetBookmark(older, true)

	d.Save("claude", []session.Session{newer})
	if got := ids(t, d, "claude"); len(got) != 1 || got[0] != "new" {
		t.Errorf("while gone: got %v, want [new]", got)
	}

	d.Save("claude", []session.Session{older, newer})
	list, _ := d.List("claude")
	if list[0].ID != "old" || !list[0].Bookmarked {
		t.Errorf("after it returns: first = %+v, want old and bookmarked", list[0])
	}
}

func TestSave_OtherAgentUntouched(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.Save("opencode", []session.Session{{Agent: "opencode", ID: "ses_1", LastUsed: now}})

	if got := ids(t, d, "claude"); len(got) != 2 {
		t.Errorf("claude rows = %v, want 2", got)
	}
	if got := ids(t, d, "opencode"); len(got) != 1 {
		t.Errorf("opencode rows = %v, want 1", got)
	}
}

func TestRemove(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.SetBookmark(older, true)

	if err := d.Remove(older); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, d, "claude"); len(got) != 1 || got[0] != "new" {
		t.Errorf("got %v, want [new]", got)
	}

	d.Save("claude", []session.Session{older, newer})
	if list, _ := d.List("claude"); list[1].Bookmarked {
		t.Error("Remove should also drop the bookmark")
	}
}

func TestFind(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{
		{Agent: "claude", ID: "2b3f3fde-1111", LastUsed: now},
		{Agent: "claude", ID: "2b3f9999-2222", LastUsed: now},
	})
	d.Save("opencode", []session.Session{{Agent: "opencode", ID: "ses_eb20", LastUsed: now}})

	cases := []struct {
		prefix string
		want   int
	}{
		{"2b3f3", 1},
		{"2b3f", 2},
		{"ses_", 1},
		{"zzz", 0},
	}
	for _, c := range cases {
		got, err := d.Find(c.prefix)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != c.want {
			t.Errorf("Find(%q) = %d matches, want %d", c.prefix, len(got), c.want)
		}
	}

	d.SetBookmark(session.Session{Agent: "opencode", ID: "ses_eb20"}, true)
	if got, _ := d.Find("ses_"); len(got) != 1 || !got[0].Bookmarked {
		t.Errorf("Find should show the bookmark: %+v", got)
	}
}

type fakeSource struct{ list []session.Session }

func (fakeSource) Name() string                           { return "claude" }
func (f fakeSource) List() ([]session.Session, error)     { return f.list, nil }
func (fakeSource) ResumeCommand(session.Session) []string { return nil }
func (fakeSource) Delete(session.Session, string) error   { return nil }

func TestRefresh(t *testing.T) {
	d := openTest(t)
	if err := d.Refresh(fakeSource{list: []session.Session{newer}}); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, d, "claude"); len(got) != 1 || got[0] != "new" {
		t.Errorf("got %v, want [new]", got)
	}
}

func TestSave_EmptyListKeepsRows(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})

	if err := d.Save("claude", nil); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, d, "claude"); len(got) != 2 {
		t.Errorf("an empty list wiped the rows: got %v", got)
	}
}

// oldFile makes an index file the way burrow-G 0.0.1 did. newer marks it as
// made by a burrow-G newer than this one.
func oldFile(t *testing.T, newer bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sessions.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, err = conn.Exec(`CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 0)`)
	if err == nil && newer {
		_, err = conn.Exec(`CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
			INSERT INTO meta VALUES ('version', '999.0.0')`)
	}
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpen_PatchesOldFile(t *testing.T) {
	d, err := Open(oldFile(t, false))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	var n int
	d.sql.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'bookmarked'`).Scan(&n)
	if n != 0 {
		t.Error("Open should have run the patches and dropped bookmarked")
	}
	if got := ids(t, d, "claude"); len(got) != 1 {
		t.Errorf("rows = %v, want [kept]", got)
	}
}

func TestOpen_RefusesNewerFile(t *testing.T) {
	path := oldFile(t, true)
	if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "newer burrow-G") || !strings.Contains(err.Error(), path) {
		t.Errorf("err = %v, want a 'newer burrow-G' error naming the file", err)
	}
}

func columnsOf(t *testing.T, path string) string {
	t.Helper()
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var cols string
	if err := conn.QueryRow(`SELECT group_concat(name) FROM pragma_table_info('sessions')`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	return cols
}

func TestOpen_BacksUpBeforePatching(t *testing.T) {
	path := oldFile(t, false)
	os.WriteFile(path+".bak", []byte("an older backup"), 0o644)

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	if cols := columnsOf(t, path+".bak"); !strings.Contains(cols, "bookmarked") {
		t.Errorf("backup columns = %q, want the file as it was before patching", cols)
	}
}

func TestOpen_NoBackupWhenNothingToPatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	for range 2 { // brand new, then already up to date
		d, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		d.Close()
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Error("no patch ran, so there should be no backup")
	}
}

func TestOpen_EmptyFileIsNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	os.WriteFile(path, nil, 0o644)

	d, err := Open(path)
	if err != nil {
		t.Fatalf("an empty file should open as a new one: %v", err)
	}
	d.Close()
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Error("an empty file has nothing to back up")
	}
}

func TestOpen_NewFileGetsLatestVersion(t *testing.T) {
	d := openTest(t)
	version, err := patches.GetPatchVersion(context.Background(), d.sql)
	if err != nil {
		t.Fatal(err)
	}
	if version != patches.LatestVersion() {
		t.Errorf("version = %s, want %s", version, patches.LatestVersion())
	}
}
