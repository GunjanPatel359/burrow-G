// Package copilot reads GitHub Copilot CLI sessions from its SQLite database.
package copilot

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads sessions from DBPath, usually ~/.copilot/session-store.db.
type Source struct {
	DBPath string
}

func (Source) Name() string { return "copilot" }

func (c Source) List() ([]session.Session, error) {
	if _, err := os.Stat(c.DBPath); os.IsNotExist(err) {
		return nil, session.ErrMissing
	} else if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", "file:"+c.DBPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, cwd, summary, updated_at FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []session.Session
	for rows.Next() {
		var id, cwd, updated string
		var summary sql.NullString
		if err := rows.Scan(&id, &cwd, &summary, &updated); err != nil {
			return nil, err
		}

		title := summary.String
		if title == "" {
			title = "(untitled)"
		}
		lastUsed, _ := time.Parse(time.RFC3339Nano, updated)

		list = append(list, session.Session{
			Agent:    "copilot",
			ID:       id,
			Title:    title,
			Dir:      cwd,
			LastUsed: lastUsed,
		})
	}
	return list, rows.Err()
}

func (Source) ResumeCommand(s session.Session) []string {
	return []string{"copilot", "--resume", s.ID}
}

func (Source) Delete(s session.Session, trashDir string) error {
	return errors.New("delete not supported for copilot yet")
}

// Export reads the session's event log itself: Copilot has no export
// command. The reader is in export.go.
func (c Source) Export(s session.Session, w *portable.Writer) error {
	path, err := c.eventsFile(s.ID)
	if err != nil {
		return err
	}
	h, err := c.header(s.ID)
	if err != nil {
		return err
	}
	if h.Created, h.Updated, err = times(path); err != nil {
		return err
	}
	if err := w.Header(h); err != nil {
		return err
	}
	return writeEvents(path, w)
}
