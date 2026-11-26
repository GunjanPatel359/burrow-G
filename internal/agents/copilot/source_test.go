package copilot

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

func TestList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-store.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE sessions (id TEXT, cwd TEXT, summary TEXT, updated_at TEXT);
		INSERT INTO sessions VALUES ('a1', '/tmp/x', 'Review App', '2026-08-20T11:45:47.227Z');
		INSERT INTO sessions VALUES ('a2', '/tmp/y', NULL, '2026-07-07T14:59:47.556Z');
	`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	src := Source{DBPath: path}
	if src.Name() != "copilot" {
		t.Errorf("Name() = %q, want copilot", src.Name())
	}

	got, err := src.List()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]session.Session{}
	for _, s := range got {
		byID[s.ID] = s
	}

	a1 := byID["a1"]
	wantTime := time.Date(2026, 8, 20, 11, 45, 47, 227_000_000, time.UTC)
	if a1.Agent != "copilot" || a1.Title != "Review App" || a1.Dir != "/tmp/x" || !a1.LastUsed.Equal(wantTime) {
		t.Errorf("a1 = %+v", a1)
	}
	if byID["a2"].Title != "(untitled)" {
		t.Errorf("a2 title = %q, want (untitled)", byID["a2"].Title)
	}
}

func TestList_MissingDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.db")

	if _, err := (Source{DBPath: path}).List(); !errors.Is(err, session.ErrMissing) {
		t.Errorf("err = %v, want session.ErrMissing", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("List must not create the DB file")
	}
}

func TestResumeCommand(t *testing.T) {
	got := Source{}.ResumeCommand(session.Session{ID: "a1"})
	want := []string{"copilot", "--resume", "a1"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDelete_NotSupported(t *testing.T) {
	if err := (Source{}).Delete(session.Session{ID: "a1"}, t.TempDir()); err == nil {
		t.Error("want an error: delete is not supported for copilot yet")
	}
}
