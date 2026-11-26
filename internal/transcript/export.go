package transcript

import (
	"fmt"
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
// The id comes from agent data, so one that would leave the folder is refused.
func SaveExport(home string, s session.Session, md string) (string, error) {
	path := ExportPath(home, s)
	if name := s.Agent + "-" + s.ID + ".md"; filepath.Base(name) != name || s.ID == "" {
		return "", fmt.Errorf("can't name an export file after session id %q", s.ID)
	}
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
// hold private conversation. An existing file is narrowed to 0600 before
// anything is written into it; a device such as /dev/stdout is left as is.
func WritePrivate(path, data string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err == nil && info.Mode().IsRegular() {
		err = f.Chmod(0o600)
	}
	if err == nil {
		_, err = f.WriteString(data)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
