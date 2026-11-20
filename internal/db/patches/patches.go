// Package patches brings index files made by older versions of burrow-G up to
// date. Each release that changes storage gets one patch file (patch0_0_2.go,
// …) and one line in chain. Never change a released patch; fixes go in the
// next release's patch.
package patches

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Patch is one release's changes. Invoke writes them with exec.
type Patch interface {
	Version() string // the burrow-G release that needs it, e.g. "0.0.2"
	Invoke() error
}

// chain is every patch in release order.
var chain = []Patch{
	patch0_0_2{},
}

// tx is the transaction of the patch that is running.
var (
	mu sync.Mutex
	tx *sql.Tx
)

// exec runs a statement inside the running patch's transaction.
func exec(query string, args ...any) error {
	_, err := tx.Exec(query, args...)
	return err
}

// base is the version of a file from before versions were stored.
const base = "0.0.1"

// From returns the version the file at conn is on, "" for a new file.
func From(conn *sql.DB) (string, error) { return from(conn, chain) }

// Needed reports whether a file on this version needs patches.
func Needed(version string) bool { return needed(chain, version) }

// Run patches a file on this version, or only stores the latest version on a
// new one, since its tables are already up to date.
func Run(conn *sql.DB, version string) error { return run(conn, chain, version) }

func from(conn *sql.DB, chain []Patch) (string, error) {
	if ok, err := hasTable(conn, "meta"); err != nil || !ok {
		if ok, err := hasTable(conn, "sessions"); err != nil || !ok {
			return "", err
		}
		return base, nil
	}
	var v string
	if err := conn.QueryRow(`SELECT value FROM meta WHERE key = 'version'`).Scan(&v); err != nil {
		return "", err
	}
	if newer(v, latest(chain)) {
		return "", fmt.Errorf("the index was made by a newer burrow-G (version %s, this one knows up to %s), please upgrade burrow-G", v, latest(chain))
	}
	return v, nil
}

func needed(chain []Patch, version string) bool {
	return version != "" && newer(latest(chain), version)
}

func run(conn *sql.DB, chain []Patch, version string) error {
	if version == "" {
		return apply(conn, latest(chain), nil)
	}
	for _, p := range chain {
		if !newer(p.Version(), version) {
			continue
		}
		if err := apply(conn, p.Version(), p); err != nil {
			return fmt.Errorf("patch %s: %w", p.Version(), err)
		}
	}
	return nil
}

// apply runs a patch and stores its version in one transaction, so a patch
// that fails changes nothing.
func apply(conn *sql.DB, version string, p Patch) error {
	mu.Lock()
	defer mu.Unlock()

	var err error
	if tx, err = conn.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()

	if p != nil {
		if err := p.Invoke(); err != nil {
			return err
		}
	}
	err = exec(`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT OR REPLACE INTO meta VALUES ('version', ?)`, version)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func latest(chain []Patch) string {
	if len(chain) == 0 {
		return base
	}
	return chain[len(chain)-1].Version()
}

func hasTable(conn *sql.DB, name string) (bool, error) {
	var n int
	err := conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&n)
	return n > 0, err
}

// newer reports whether version a comes after b, e.g. 0.0.10 after 0.0.9.
func newer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		x, y := part(as, i), part(bs, i)
		if x != y {
			return x > y
		}
	}
	return false
}

func part(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[i])
	return n
}
