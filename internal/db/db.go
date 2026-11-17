// Package db is burrow-G's own SQLite database.
package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// sessions is an index of the agents' files and can always be rebuilt.
// bookmarks only exists here, so refreshing never touches it.
const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	agent     TEXT NOT NULL,
	id        TEXT NOT NULL,
	title     TEXT NOT NULL,
	dir       TEXT NOT NULL,
	last_used INTEGER NOT NULL,
	PRIMARY KEY (agent, id)
);
CREATE TABLE IF NOT EXISTS bookmarks (
	agent TEXT NOT NULL,
	id    TEXT NOT NULL,
	PRIMARY KEY (agent, id)
);`

type DB struct {
	sql *sql.DB
}

// Open opens the database at path, creating it and its tables if needed.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1) // SQLite allows one writer at a time

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}
	if err := moveBookmarks(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{sql: conn}, nil
}

// moveBookmarks moves bookmarks out of the sessions table, where burrow-G
// 0.0.1 kept them, into the bookmarks table, then drops the old column.
func moveBookmarks(conn *sql.DB) error {
	var n int
	err := conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'bookmarked'`).Scan(&n)
	if err != nil || n == 0 {
		return err
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR IGNORE INTO bookmarks (agent, id) SELECT agent, id FROM sessions WHERE bookmarked = 1`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE sessions DROP COLUMN bookmarked`); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) Close() error {
	return d.sql.Close()
}
