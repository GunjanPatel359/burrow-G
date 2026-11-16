package helpers

import (
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	if got := DataDir("/home/u"); got != "/home/u/.local/share/burrow-g" {
		t.Errorf("DataDir = %q", got)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if !Exists(dir) {
		t.Error("Exists(existing dir) = false")
	}
	if Exists(filepath.Join(dir, "nope")) {
		t.Error("Exists(missing) = true")
	}
}
