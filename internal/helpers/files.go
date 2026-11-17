package helpers

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is where burrow-G keeps its own data.
func DataDir(home string) string {
	return dataDir(runtime.GOOS, home, os.Getenv)
}

func dataDir(goos, home string, getenv func(string) string) string {
	if goos != "windows" {
		return filepath.Join(home, ".local", "share", "burrow-g")
	}
	if dir := getenv("LOCALAPPDATA"); dir != "" {
		return filepath.Join(dir, "burrow-g")
	}
	return filepath.Join(home, "AppData", "Local", "burrow-g")
}

// Exists reports whether path is there.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
