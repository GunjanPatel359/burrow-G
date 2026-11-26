package qoder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Transcriber = Source{}

func TestTranscript(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "projects", "-tmp-proj")
	copyFile(t, "testdata/transcript/main.jsonl", filepath.Join(proj, "q1.jsonl"))
	if err := os.Chmod(proj, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(proj, 0o755) })

	got, err := Source{Home: home}.Transcript(session.Session{Agent: "qoder", ID: "q1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 4 || got.Skipped != 1 {
		t.Fatalf("got %d messages, %d skipped: %+v", len(got.Messages), got.Skipped, got.Messages)
	}
	if m := got.Messages[1]; m.Text != "Running them now." || len(m.Tools) != 1 || m.Tools[0] != "Bash go test ./..." {
		t.Errorf("assistant = %+v", m)
	}
}

func TestStartCommand(t *testing.T) {
	got := (Source{}).StartCommand("/tmp", "hi")
	if len(got) != 3 || got[0] != "qodercli" || got[1] != "-i" || got[2] != "hi" {
		t.Errorf("got %q", got)
	}
}
