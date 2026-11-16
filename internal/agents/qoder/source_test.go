package qoder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

func TestParseFile(t *testing.T) {
	cases := []struct {
		file, title, cwd string
	}{
		{"testdata/titled.jsonl", "Review phase two", "/tmp/q-proj"},
		{"testdata/untitled.jsonl", "check the build", "/tmp/q-other"},
		{"testdata/empty.jsonl", "", ""},
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

func TestList(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "projects", "-tmp-q-proj")
	copyFile(t, "testdata/titled.jsonl", filepath.Join(proj, "q1.jsonl"))
	copyFile(t, "testdata/empty.jsonl", filepath.Join(proj, "q2.jsonl"))

	used := time.Date(2026, 11, 13, 10, 0, 0, 0, time.Local)
	if err := os.Chtimes(filepath.Join(proj, "q1.jsonl"), used, used); err != nil {
		t.Fatal(err)
	}

	src := Source{Home: home}
	if src.Name() != "qoder" {
		t.Errorf("Name() = %q, want qoder", src.Name())
	}

	got, err := src.List()
	if err != nil {
		t.Fatal(err)
	}
	want := session.Session{Agent: "qoder", ID: "q1", Title: "Review phase two", Dir: "/tmp/q-proj", LastUsed: used}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v\nwant [%+v]", got, want)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
