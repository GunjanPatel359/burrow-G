package transcript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestWritePrivate_NarrowsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.md")
	if err := os.WriteFile(path, []byte("old and longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivate(path, "new"); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	data, _ := os.ReadFile(path)
	if info.Mode().Perm() != 0o600 || string(data) != "new" {
		t.Errorf("mode %v, content %q", info.Mode().Perm(), data)
	}
}

func TestWritePrivate_LeavesDevicesAlone(t *testing.T) {
	before, err := os.Stat(os.DevNull)
	if err != nil {
		t.Skip(err)
	}
	if err := WritePrivate(os.DevNull, "x"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(os.DevNull)
	if after.Mode() != before.Mode() {
		t.Errorf("%s mode changed from %v to %v", os.DevNull, before.Mode(), after.Mode())
	}
}

func TestSaveExport(t *testing.T) {
	home := t.TempDir()
	path, err := SaveExport(home, session.Session{Agent: "claude", ID: "abc"}, "md")
	if err != nil || path != filepath.Join(home, ".local", "share", "burrow-g", "exports", "claude-abc.md") {
		t.Fatalf("path %q, err %v", path, err)
	}
	dir, _ := os.Stat(filepath.Dir(path))
	if dir.Mode().Perm() != 0o700 {
		t.Errorf("folder mode %v, want 700", dir.Mode().Perm())
	}
	for _, id := range []string{"../../escape", "a/b", ""} {
		if _, err := SaveExport(home, session.Session{Agent: "claude", ID: id}, "md"); err == nil {
			t.Errorf("id %q: want an error", id)
		}
	}
}

func TestWritePrivate_ShorterContentReplacesLonger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.md")
	if err := WritePrivate(path, "a much longer first export"); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivate(path, "short"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "short" {
		t.Errorf("content = %q, want the old text gone", data)
	}
}
