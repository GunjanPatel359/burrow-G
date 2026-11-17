// Package opencode reads OpenCode sessions from its SQLite database.
package opencode

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads sessions from DBPath, usually ~/.local/share/opencode/opencode.db.
type Source struct {
	DBPath string
}

func (Source) Name() string { return "opencode" }

func (o Source) List() ([]session.Session, error) {
	if _, err := os.Stat(o.DBPath); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", db.ReadOnly(o.DBPath))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.Query(`SELECT id, title, directory, time_updated FROM session WHERE parent_id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []session.Session
	for rows.Next() {
		var s session.Session
		var updated int64
		if err := rows.Scan(&s.ID, &s.Title, &s.Dir, &updated); err != nil {
			return nil, err
		}
		s.Agent = "opencode"
		s.LastUsed = time.UnixMilli(updated)
		list = append(list, s)
	}
	return list, rows.Err()
}

func (Source) ResumeCommand(s session.Session) []string {
	return []string{"opencode", "--session", s.ID}
}

func (Source) Delete(s session.Session, trashDir string) error {
	return errors.New("not implemented")
}
