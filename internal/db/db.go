// Package db is burrow-G's own SQLite database.
package db

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	conn, err := sql.Open("sqlite", fileURI(path, filepath.Separator, ""))
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1) // SQLite allows one writer at a time

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{sql: conn}, nil
}

func (d *DB) Close() error {
	return d.sql.Close()
}

// ReadOnly returns a SQLite URI that opens path without ever writing to it.
func ReadOnly(path string) string {
	return fileURI(path, filepath.Separator, "mode=ro")
}

// fileURI turns a file path into a SQLite URI, so any folder name works.
func fileURI(path string, sep rune, query string) string {
	p := strings.ReplaceAll(path, string(sep), "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // C:/x → /C:/x
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: query}
	return u.String()
}
