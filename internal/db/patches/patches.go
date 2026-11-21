package patches

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type PatchContext struct {
	Tx *sql.Tx
}

type Patch interface {
	Version() string
	Invoke(ctx context.Context, pct PatchContext) error
}

var chain = []Patch{
	patch002{},
}

// LatestVersion returns the version of the last registered patch.
func LatestVersion() string {
	if len(chain) == 0 {
		return "0.0.1"
	}
	return chain[len(chain)-1].Version()
}

// Run applies newer patches using an already-prepared database connection.
func Run(ctx context.Context, conn *sql.DB) error {
	version, err := GetPatchVersion(ctx, conn)
	if err != nil {
		return err
	}
	for _, p := range chain {
		if !VersionCheck(p.Version(), version) {
			continue
		}
		if err := executePatch(ctx, conn, p); err != nil {
			return fmt.Errorf("patch %s: %w", p.Version(), err)
		}
		version = p.Version()
	}
	return nil
}

// executePatch commits a patch and its version in one transaction.
func executePatch(ctx context.Context, conn *sql.DB, p Patch) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := p.Invoke(ctx, PatchContext{Tx: tx}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT OR REPLACE INTO meta (key, value) VALUES ('version', ?)`, p.Version()); err != nil {
		return err
	}
	return tx.Commit()
}

// GetPatchVersion returns the stored version, defaulting to 0.0.1 if absent.
func GetPatchVersion(ctx context.Context, conn *sql.DB) (string, error) {
	var version string
	err := conn.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'version'`).Scan(&version)
	if err == nil {
		return version, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "0.0.1", nil
	}
	if strings.Contains(err.Error(), "no such table: meta") {
		return "0.0.1", nil
	}
	return "", err
}

// VersionCheck reports whether version a is greater than version b.
func VersionCheck(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}
