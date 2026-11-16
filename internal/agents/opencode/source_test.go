package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

func TestList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE session (id TEXT, title TEXT, directory TEXT, parent_id TEXT, time_updated INTEGER);
		INSERT INTO session VALUES ('ses_1', 'Fix auth', '/tmp/p', NULL, 1794402177784);
		INSERT INTO session VALUES ('ses_2', 'Child task', '/tmp/p', 'ses_1', 1794402177999);
	`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	src := Source{DBPath: path}
	if src.Name() != "opencode" {
		t.Errorf("Name() = %q, want opencode", src.Name())
	}

	got, err := src.List()
	if err != nil {
		t.Fatal(err)
	}
	want := session.Session{Agent: "opencode", ID: "ses_1", Title: "Fix auth", Dir: "/tmp/p", LastUsed: time.UnixMilli(1794402177784)}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v\nwant [%+v]", got, want)
	}
}

func TestList_MissingDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.db")

	if _, err := (Source{DBPath: path}).List(); err == nil {
		t.Error("want an error for a missing DB")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("List must not create the DB file")
	}
}
