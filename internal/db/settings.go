package db

import (
	"database/sql"
	"errors"
)

// Setting returns the value stored under key, or "" if it is unset.
func (d *DB) Setting(key string) (string, error) {
	var value string
	err := d.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SetSetting stores value under key, replacing any earlier value.
func (d *DB) SetSetting(key, value string) error {
	_, err := d.sql.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}
