// Package db is burrow-G's own SQLite database.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/GunjanPatel359/burrow-G/internal/db/patches"
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

// Open opens the database at path, creating it and its tables if needed, and
// brings an older file up to date.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1) // SQLite allows one writer at a time

	if err := prepare(conn, path); err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &DB{sql: conn}, nil
}

// prepare backs up a file that needs patches, before anything changes it,
// then creates missing tables and runs the patches.
func prepare(conn *sql.DB, path string) error {
	version, err := patches.From(conn)
	if err != nil {
		return err
	}
	if patches.Needed(version) {
		if err := backup(conn, path+".bak"); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
	}
	if _, err := conn.Exec(schema); err != nil {
		return err
	}
	return patches.Run(conn, version)
}

// backup writes a copy of the database to path, replacing an older backup.
func backup(conn *sql.DB, path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	_, err := conn.Exec(`VACUUM INTO ?`, path)
	return err
}

func (d *DB) Close() error {
	return d.sql.Close()
}
