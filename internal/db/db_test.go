package db

import (
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

func TestSave_DropsBookmarksOfGoneSessions(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.SetBookmark(older, true)
	d.SetBookmark(session.Session{Agent: "codex", ID: "other"}, true)

	d.Save("claude", []session.Session{newer})
	var n int
	d.sql.QueryRow(`SELECT COUNT(*) FROM bookmarks`).Scan(&n)
	if n != 1 {
		t.Errorf("bookmarks = %d, want only codex's left", n)
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

type fakeSource struct {
	list []session.Session
	err  error
}

func (fakeSource) Name() string                           { return "claude" }
func (f fakeSource) List() ([]session.Session, error)     { return f.list, f.err }
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

func TestRefresh_MissingFolderKeepsRows(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})
	d.SetBookmark(older, true)

	if err := d.Refresh(fakeSource{err: session.ErrMissing}); err != nil {
		t.Errorf("a missing folder is not an error: %v", err)
	}
	if list, _ := d.List("claude"); len(list) != 2 || !list[0].Bookmarked {
		t.Errorf("a missing folder changed the list: %+v", list)
	}
}

func TestRefresh_EmptyListClears(t *testing.T) {
	d := openTest(t)
	d.Save("claude", []session.Session{older, newer})

	if err := d.Refresh(fakeSource{}); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, d, "claude"); len(got) != 0 {
		t.Errorf("got %v, want none: every session is gone", got)
	}
}
