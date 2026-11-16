package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	cases := []struct {
		file, title, cwd string
	}{
		{"testdata/titled.jsonl", "First title", "/tmp/proj-a"},
		{"testdata/untitled.jsonl", "fix the login bug please", "/tmp/proj-b"},
		{"testdata/empty.jsonl", "", ""},
		{"testdata/renamed.jsonl", "My rename", "/tmp/proj-c"},
		{"testdata/meta.jsonl", "real question here", "/tmp/meta"},
		{"testdata/command.jsonl", "/about", "/tmp/cmd"},
		{"testdata/stdout.jsonl", "what does this mean", "/tmp/out"},
	}

	for _, c := range cases {
		title, cwd, err := parseFile(c.file)
		if err != nil {
			t.Fatalf("%s: %v", c.file, err)
		}
		if title != c.title || cwd != c.cwd {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.file, title, cwd, c.title, c.cwd)
		}
	}
}

func TestParseFile_HugeLine(t *testing.T) {
	big := strings.Repeat("a", 5_000_000)
	data := `{"type":"user","cwd":"/tmp/big","message":{"role":"user","content":"` + big + `"}}` + "\n" +
		`{"type":"ai-title","aiTitle":"Big"}` + "\n"

	path := filepath.Join(t.TempDir(), "big.jsonl")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	title, cwd, err := parseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Big" || cwd != "/tmp/big" {
		t.Errorf("got (%q, %q), want (%q, %q)", title, cwd, "Big", "/tmp/big")
	}
}
