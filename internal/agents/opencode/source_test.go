package opencode

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

// columns are the ones List reads; real OpenCode tables have many more.
const columns = `(id TEXT, title TEXT, directory TEXT, parent_id TEXT, time_updated INTEGER)`

// newDB writes an opencode.db made of the given SQL and returns its path.
func newDB(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(script); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestName(t *testing.T) {
	if got := (Source{}).Name(); got != "opencode" {
		t.Errorf("Name() = %q, want opencode", got)
	}
}

func TestList(t *testing.T) {
	fixAuth := session.Session{Agent: "opencode", ID: "ses_1", Title: "Fix auth", Dir: "/tmp/p", LastUsed: time.UnixMilli(1794402177784)}

	tests := []struct {
		name   string
		script string
		want   []session.Session
	}{
		{
			// OpenCode renamed the table in newer versions.
			name: "session_v2 table",
			script: `CREATE TABLE session_v2 ` + columns + `;
				INSERT INTO session_v2 VALUES ('ses_1', 'Fix auth', '/tmp/p', NULL, 1794402177784);
				INSERT INTO session_v2 VALUES ('ses_2', 'Child task', '/tmp/p', 'ses_1', 1794402177999);`,
			want: []session.Session{fixAuth},
		},
		{
			name: "older session table",
			script: `CREATE TABLE session ` + columns + `;
				INSERT INTO session VALUES ('ses_1', 'Fix auth', '/tmp/p', NULL, 1794402177784);
				INSERT INTO session VALUES ('ses_2', 'Child task', '/tmp/p', 'ses_1', 1794402177999);`,
			want: []session.Session{fixAuth},
		},
		{
			name: "both tables: session_v2 wins",
			script: `CREATE TABLE session ` + columns + `;
				INSERT INTO session VALUES ('old_1', 'Stale copy', '/tmp/old', NULL, 1);
				CREATE TABLE session_v2 ` + columns + `;
				INSERT INTO session_v2 VALUES ('ses_1', 'Fix auth', '/tmp/p', NULL, 1794402177784);`,
			want: []session.Session{fixAuth},
		},
		{
			// A session OpenCode hasn't named yet has a NULL title.
			name: "untitled session",
			script: `CREATE TABLE session_v2 ` + columns + `;
				INSERT INTO session_v2 VALUES ('ses_3', NULL, '/tmp/p', NULL, 1794402177784);`,
			want: []session.Session{{Agent: "opencode", ID: "ses_3", Title: "(untitled)", Dir: "/tmp/p", LastUsed: time.UnixMilli(1794402177784)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Source{DBPath: newDB(t, tt.script)}.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %+v\nwant %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %+v\nwant %+v", got[i], tt.want[i])
				}
			}
		})
	}
}

func TestList_NoSessionTable(t *testing.T) {
	path := newDB(t, `CREATE TABLE project (id TEXT)`)

	_, err := Source{DBPath: path}.List()
	if err == nil || errors.Is(err, session.ErrMissing) {
		t.Fatalf("err = %v, want an unsupported-database error", err)
	}
	if !strings.Contains(err.Error(), "unsupported database schema") {
		t.Errorf("err = %q, want an unsupported database schema error", err)
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
