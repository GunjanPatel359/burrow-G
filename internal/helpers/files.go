package helpers

import (
	"os"
	"path/filepath"
)

// DataDir is where burrow-G keeps its own data.
func DataDir(home string) string {
	return filepath.Join(home, ".local", "share", "burrow-g")
}

// Exists reports whether path is there.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
