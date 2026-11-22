// Package db is burrow-G's own SQLite database.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/GunjanPatel359/burrow-G/internal/db/patches"
)

// meta holds the file's version. sessions is an index of the agents' files
// and can always be rebuilt. bookmarks only exists here, so refreshing never
// touches it.
const schema = `
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
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
);
CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`

type DB struct {
	sql *sql.DB
}

// Open opens the database at path, creating it and its tables if needed, and
// brings an older file up to date. version is this burrow-G's release.
func Open(path, version string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1) // SQLite allows one writer at a time

	if err := prepare(conn, path, version); err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &DB{sql: conn}, nil
}

// prepare patches an older file, after a backup, then creates missing tables
// and stores this release's version.
func prepare(conn *sql.DB, path, latest string) error {
	ctx := context.Background()
	version, err := patches.GetPatchVersion(ctx, conn)
	if err != nil {
		return err
	}
	if patches.VersionCheck(version, latest) {
		return fmt.Errorf("the index was made by a newer burrow-G (version %s, this one knows up to %s), please upgrade burrow-G", version, latest)
	}
	if patches.VersionCheck(patches.LatestVersion(), version) {
		if err := backup(conn, path+".bak"); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
	}
	if err := patches.Run(ctx, conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, schema); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `INSERT OR REPLACE INTO meta (key, value) VALUES ('version', ?)`, latest)
	return err
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
