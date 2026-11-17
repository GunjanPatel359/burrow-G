package helpers

import (
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	none := func(string) string { return "" }
	local := func(k string) string {
		if k == "LOCALAPPDATA" {
			return `C:\Users\u\AppData\Local`
		}
		return ""
	}

	cases := []struct {
		goos   string
		getenv func(string) string
		want   string
	}{
		{"linux", local, filepath.Join("/home/u", ".local", "share", "burrow-g")},
		{"darwin", none, filepath.Join("/home/u", ".local", "share", "burrow-g")},
		{"windows", local, filepath.Join(`C:\Users\u\AppData\Local`, "burrow-g")},
		{"windows", none, filepath.Join("/home/u", "AppData", "Local", "burrow-g")},
	}
	for _, c := range cases {
		if got := dataDir(c.goos, "/home/u", c.getenv); got != c.want {
			t.Errorf("%s: dataDir = %q, want %q", c.goos, got, c.want)
		}
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
