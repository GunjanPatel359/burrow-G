// Package db is burrow-G's own SQLite database.
package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	agent      TEXT NOT NULL,
	id         TEXT NOT NULL,
	title      TEXT NOT NULL,
	dir        TEXT NOT NULL,
	last_used  INTEGER NOT NULL,
	bookmarked INTEGER NOT NULL DEFAULT 0,
	owner      TEXT NOT NULL DEFAULT '',
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
	if err := addOwner(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{sql: conn}, nil
}

func (d *DB) Close() error {
	return d.sql.Close()
}

// addOwner adds the owner column to databases made by burrow-G 0.0.1.
func addOwner(conn *sql.DB) error {
	var n int
	err := conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'owner'`).Scan(&n)
	if err != nil || n > 0 {
		return err
	}
	_, err = conn.Exec(`ALTER TABLE sessions ADD COLUMN owner TEXT NOT NULL DEFAULT ''`)
	return err
}
