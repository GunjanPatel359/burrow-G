package db

import (
	"database/sql"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Save replaces one agent's sessions. An empty list changes nothing, so a
// folder that is missing for a moment can't wipe the list.
func (d *DB) Save(agent string, list []session.Session) error {
	if len(list) == 0 {
		return nil
	}
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE agent = ?`, agent); err != nil {
		return err
	}
	for _, s := range list {
		_, err := tx.Exec(
			`INSERT INTO sessions (agent, id, title, dir, last_used) VALUES (?, ?, ?, ?, ?)`,
			agent, s.ID, s.Title, s.Dir, s.LastUsed.UnixMilli(),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// withBookmark selects a session plus whether it is bookmarked.
const withBookmark = `
	SELECT s.agent, s.id, s.title, s.dir, s.last_used, b.id IS NOT NULL
	FROM sessions s LEFT JOIN bookmarks b ON b.agent = s.agent AND b.id = s.id`

// List returns one agent's sessions, bookmarked first, then newest first.
func (d *DB) List(agent string) ([]session.Session, error) {
	rows, err := d.sql.Query(withBookmark+`
		WHERE s.agent = ? ORDER BY b.id IS NOT NULL DESC, s.last_used DESC`, agent)
	if err != nil {
		return nil, err
	}
	return scan(rows)
}

// Find returns every session whose id starts with prefix, across all agents.
func (d *DB) Find(prefix string) ([]session.Session, error) {
	rows, err := d.sql.Query(withBookmark+`
		WHERE substr(s.id, 1, length(?)) = ?`, prefix, prefix)
	if err != nil {
		return nil, err
	}
	return scan(rows)
}

func scan(rows *sql.Rows) ([]session.Session, error) {
	defer rows.Close()
	var list []session.Session
	for rows.Next() {
		var s session.Session
		var lastUsed int64
		if err := rows.Scan(&s.Agent, &s.ID, &s.Title, &s.Dir, &lastUsed, &s.Bookmarked); err != nil {
			return nil, err
		}
		s.LastUsed = time.UnixMilli(lastUsed)
		list = append(list, s)
	}
	return list, rows.Err()
}

// Counts returns how many sessions each agent has.
func (d *DB) Counts() (map[string]int, error) {
	rows, err := d.sql.Query(`SELECT agent, COUNT(*) FROM sessions GROUP BY agent`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var agent string
		var n int
		if err := rows.Scan(&agent, &n); err != nil {
			return nil, err
		}
		counts[agent] = n
	}
	return counts, rows.Err()
}

// SetBookmark adds or removes a session's bookmark.
func (d *DB) SetBookmark(s session.Session, on bool) error {
	query := `DELETE FROM bookmarks WHERE agent = ? AND id = ?`
	if on {
		query = `INSERT OR IGNORE INTO bookmarks (agent, id) VALUES (?, ?)`
	}
	_, err := d.sql.Exec(query, s.Agent, s.ID)
	return err
}

// Remove forgets a session and its bookmark.
func (d *DB) Remove(s session.Session) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"sessions", "bookmarks"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE agent = ? AND id = ?`, s.Agent, s.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Refresh reloads one agent's sessions from its files into the database.
func (d *DB) Refresh(src session.Source) error {
	list, err := src.List()
	if err != nil {
		return err
	}
	return d.Save(src.Name(), list)
}
