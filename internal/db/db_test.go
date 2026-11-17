package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

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

func TestSetBookmark_MovesToTop(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})

	if err := d.SetBookmark(older, true); err != nil {
		t.Fatal(err)
	}

	list, _ := d.List("claude")
	if list[0].ID != "old" || !list[0].Bookmarked {
		t.Errorf("first = %+v, want old and bookmarked", list[0])
	}
}

func TestSave_KeepsBookmarksAndDropsGone(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.SetBookmark(older, true)

	fresh := session.Session{Agent: "claude", ID: "fresh", Title: "Fresh", Dir: "/tmp/c", LastUsed: now}
	if err := d.Save("claude", []session.Session{older, fresh}); err != nil {
		t.Fatal(err)
	}

	got := ids(t, d, "claude")
	if len(got) != 2 || got[0] != "old" || got[1] != "fresh" {
		t.Errorf("got %v, want [old fresh]", got)
	}
	list, _ := d.List("claude")
	if !list[0].Bookmarked {
		t.Error("bookmark on 'old' was lost")
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

	if err := d.Remove(older); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, d, "claude"); len(got) != 1 || got[0] != "new" {
		t.Errorf("got %v, want [new]", got)
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

func TestOwner_SavedListedFound(t *testing.T) {
	d := openTest(t)
	s := older
	s.Owner = "root"
	if err := d.Save("claude", []session.Session{s}); err != nil {
		t.Fatal(err)
	}

	list, err := d.List("claude")
	if err != nil || len(list) != 1 || list[0].Owner != "root" {
		t.Errorf("List = %+v, %v; want owner root", list, err)
	}
	found, err := d.Find("old")
	if err != nil || len(found) != 1 || found[0].Owner != "root" {
		t.Errorf("Find = %+v, %v; want owner root", found, err)
	}
}

func TestOpen_AddsOwnerToOldDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 0, 1)`)
	old.Close()
	if err != nil {
		t.Fatal(err)
	}

	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	list, err := d.List("claude")
	if err != nil || len(list) != 1 || !list[0].Bookmarked || list[0].Owner != "" {
		t.Errorf("List = %+v, %v; want the old row, still bookmarked, no owner", list, err)
	}
	if _, err := Open(path); err != nil {
		t.Errorf("opening again: %v", err)
	}
}
