// Package patches brings index files made by older versions of burrow-G up to
// date. Each release that changes storage gets one patch file (patch0_0_2.go,
// …) and one entry in chain. Never change a released patch; fixes go in the
// next release's patch.
package patches

import (
	"database/sql"
	"fmt"
)

type Patch struct {
	Version string // the burrow-G release that needs it, e.g. "0.0.2"
	Apply   func(tx *sql.Tx) error
}

// chain is every patch in release order. PRAGMA user_version stores how many
// of them a file has had, so each runs once per file.
var chain = []Patch{
	patch0_0_2,
}

// Run applies the patches the file at conn hasn't had yet.
func Run(conn *sql.DB) error { return run(conn, chain) }

// Pending reports whether the file at conn still needs patches.
func Pending(conn *sql.DB) (bool, error) { return pending(conn, chain) }

func pending(conn *sql.DB, chain []Patch) (bool, error) {
	done, err := applied(conn, chain)
	return done < len(chain), err
}

// applied returns how many patches the file has had. A file made by a newer
// burrow-G is an error, so it is never changed by an older one.
func applied(conn *sql.DB, chain []Patch) (int, error) {
	var done int
	if err := conn.QueryRow(`PRAGMA user_version`).Scan(&done); err != nil {
		return 0, err
	}
	if done > len(chain) {
		return 0, fmt.Errorf("the index was made by a newer burrow-G (it has %d patches, this one knows %d), please upgrade burrow-G", done, len(chain))
	}
	return done, nil
}

func run(conn *sql.DB, chain []Patch) error {
	done, err := applied(conn, chain)
	if err != nil {
		return err
	}
	for i := done; i < len(chain); i++ {
		if err := apply(conn, i+1, chain[i]); err != nil {
			return fmt.Errorf("patch %s: %w", chain[i].Version, err)
		}
	}
	return nil
}

// apply runs one patch and records it in one transaction, so a patch that
// fails changes nothing.
func apply(conn *sql.DB, n int, p Patch) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := p.Apply(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, n)); err != nil {
		return err
	}
	return tx.Commit()
}

// dropColumn removes a column if it is there. New files get every patch too,
// so patches must check before they change anything.
func dropColumn(tx *sql.Tx, table, col string) error {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&n)
	if err != nil || n == 0 {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, col))
	return err
}
