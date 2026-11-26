package transcript

import (
	"os"
	"path/filepath"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// ExportPath is where the TUI saves an export: burrow-G's own data folder,
// never an agent's.
func ExportPath(home string, s session.Session) string {
	return filepath.Join(helpers.DataDir(home), "exports", s.Agent+"-"+s.ID+".md")
}

// SaveExport writes md to ExportPath, private to the user, replacing an older export.
func SaveExport(home string, s session.Session, md string) (string, error) {
	path := ExportPath(home, s)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return path, WritePrivate(path, md)
}

// WritePrivate writes a file only its owner can read, since exports can
// hold private conversation. Chmod covers a file that already existed.
func WritePrivate(path, data string) error {
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
