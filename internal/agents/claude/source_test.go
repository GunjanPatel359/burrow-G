package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

func TestList(t *testing.T) {
	home := t.TempDir()
	projA := filepath.Join(home, "projects", "-tmp-proj-a")
	projEmpty := filepath.Join(home, "projects", "-tmp-empty")

	copyFile(t, "testdata/titled.jsonl", filepath.Join(projA, "aaa.jsonl"))
	copyFile(t, "testdata/titled.jsonl", filepath.Join(projA, "aaa", "subagents", "x.jsonl"))
	copyFile(t, "testdata/empty.jsonl", filepath.Join(projEmpty, "bbb.jsonl"))

	used := time.Date(2026, 11, 13, 10, 0, 0, 0, time.Local)
	if err := os.Chtimes(filepath.Join(projA, "aaa.jsonl"), used, used); err != nil {
		t.Fatal(err)
	}

	got, err := Source{Home: home}.List()
	if err != nil {
		t.Fatal(err)
	}

	want := session.Session{Agent: "claude", ID: "aaa", Title: "First title", Dir: "/tmp/proj-a", LastUsed: used}
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want 1: %+v", len(got), got)
	}
	if got[0] != want {
		t.Errorf("got %+v\nwant %+v", got[0], want)
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
