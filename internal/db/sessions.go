package db

import (
	"database/sql"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Save replaces one agent's sessions, keeping their bookmarks.
func (d *DB) Save(agent string, list []session.Session) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	marked, err := bookmarkedIDs(tx, agent)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE agent = ?`, agent); err != nil {
		return err
	}
	for _, s := range list {
		_, err := tx.Exec(
			`INSERT INTO sessions (agent, id, title, dir, last_used, bookmarked, owner) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			agent, s.ID, s.Title, s.Dir, s.LastUsed.UnixMilli(), marked[s.ID], s.Owner,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func bookmarkedIDs(tx *sql.Tx, agent string) (map[string]bool, error) {
	rows, err := tx.Query(`SELECT id FROM sessions WHERE agent = ? AND bookmarked = 1`, agent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	marked := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		marked[id] = true
	}
	return marked, rows.Err()
}

// List returns one agent's sessions, bookmarked first, then newest first.
func (d *DB) List(agent string) ([]session.Session, error) {
	rows, err := d.sql.Query(
		`SELECT id, title, dir, last_used, bookmarked, owner FROM sessions
		 WHERE agent = ? ORDER BY bookmarked DESC, last_used DESC`, agent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []session.Session
	for rows.Next() {
		s := session.Session{Agent: agent}
		var lastUsed int64
		if err := rows.Scan(&s.ID, &s.Title, &s.Dir, &lastUsed, &s.Bookmarked, &s.Owner); err != nil {
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

// Find returns every session whose id starts with prefix, across all agents.
func (d *DB) Find(prefix string) ([]session.Session, error) {
	rows, err := d.sql.Query(
		`SELECT agent, id, title, dir, last_used, bookmarked, owner FROM sessions
		 WHERE substr(id, 1, length(?)) = ?`, prefix, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []session.Session
	for rows.Next() {
		var s session.Session
		var lastUsed int64
		if err := rows.Scan(&s.Agent, &s.ID, &s.Title, &s.Dir, &lastUsed, &s.Bookmarked, &s.Owner); err != nil {
			return nil, err
		}
		s.LastUsed = time.UnixMilli(lastUsed)
		list = append(list, s)
	}
	return list, rows.Err()
}

func (d *DB) SetBookmark(s session.Session, on bool) error {
	_, err := d.sql.Exec(`UPDATE sessions SET bookmarked = ? WHERE agent = ? AND id = ?`, on, s.Agent, s.ID)
	return err
}

func (d *DB) Remove(s session.Session) error {
	_, err := d.sql.Exec(`DELETE FROM sessions WHERE agent = ? AND id = ?`, s.Agent, s.ID)
	return err
}

// Refresh reloads one agent's sessions from its files into the database.
func (d *DB) Refresh(src session.Source) error {
	list, err := src.List()
	if err != nil {
		return err
	}
	return d.Save(src.Name(), list)
}
