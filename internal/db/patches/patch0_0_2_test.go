package patches

import "testing"

func TestPatch0_0_2_DropsBookmarkedColumn(t *testing.T) {
	conn := rawDB(t)
	mustExec(t, conn, `CREATE TABLE sessions (agent TEXT NOT NULL, id TEXT NOT NULL, title TEXT NOT NULL,
		dir TEXT NOT NULL, last_used INTEGER NOT NULL, bookmarked INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (agent, id));
		INSERT INTO sessions VALUES ('claude', 'kept', 'Kept', '/tmp', 1, 1)`)

	if err := run(conn, []Patch{patch0_0_2{}}, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	var cols string
	conn.QueryRow(`SELECT group_concat(name) FROM pragma_table_info('sessions')`).Scan(&cols)
	if cols != "agent,id,title,dir,last_used" {
		t.Errorf("columns = %s, want bookmarked gone", cols)
	}
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1 kept", n)
	}
}
