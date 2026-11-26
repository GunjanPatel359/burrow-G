// Package opencode reads OpenCode sessions from its SQLite database.
package opencode

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads sessions from DBPath, usually ~/.local/share/opencode/opencode.db.
type Source struct {
	DBPath string
	// Run runs a command and returns its output; Export uses it for
	// OpenCode's own export. nil means export isn't available.
	Run func(name string, args ...string) ([]byte, error)
}

func (Source) Name() string { return "opencode" }

func (o Source) List() ([]session.Session, error) {
	if _, err := os.Stat(o.DBPath); os.IsNotExist(err) {
		return nil, session.ErrMissing
	} else if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", "file:"+o.DBPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Newer OpenCode versions keep sessions in session_v2, older ones in session.
	var table string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('session_v2', 'session') ORDER BY name = 'session_v2' DESC LIMIT 1`).Scan(&table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("unsupported database schema: neither session_v2 nor session table found")
	}
	if err != nil {
		return nil, err
	}
	// table is one of the two names above, never user input.
	rows, err := db.Query(`SELECT id, title, directory, time_updated FROM ` + table + ` WHERE parent_id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []session.Session
	for rows.Next() {
		var s session.Session
		var title sql.NullString
		var updated int64
		if err := rows.Scan(&s.ID, &title, &s.Dir, &updated); err != nil {
			return nil, err
		}
		s.Title = title.String
		if s.Title == "" {
			s.Title = "(untitled)"
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

// Export uses OpenCode's own structured export (export.go).
func (o Source) Export(s session.Session, w *portable.Writer) error {
	if o.Run == nil {
		return session.ErrUnsupported
	}
	return export(o.Run, s.ID, w)
}
