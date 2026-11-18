package patches

import (
	"database/sql"
	"strings"
	"testing"
)

func columns(t *testing.T, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT name FROM pragma_table_info('sessions')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		cols = append(cols, c)
	}
	return cols
}

func TestPatch0_0_2_DropsOldColumns(t *testing.T) {
	conn := rawDB(t)
	_, err := conn.Exec(`CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0,
		owner TEXT NOT NULL DEFAULT '', PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1, 'root')`)
	if err != nil {
		t.Fatal(err)
	}

	if err := run(conn, []Patch{patch0_0_2}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(columns(t, conn), ","); got != "agent,id,title,dir,last_used" {
		t.Errorf("columns = %s, want the old ones gone", got)
	}
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1 kept", n)
	}
}

func TestPatch0_0_2_NewFileUnchanged(t *testing.T) {
	conn := rawDB(t)
	_, err := conn.Exec(`CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, PRIMARY KEY (agent, id))`)
	if err != nil {
		t.Fatal(err)
	}
	if err := run(conn, []Patch{patch0_0_2}); err != nil {
		t.Errorf("a file that is already up to date: %v", err)
	}
}
