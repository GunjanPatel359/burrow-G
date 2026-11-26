package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

var _ session.Transcriber = Source{}

func TestTranscript(t *testing.T) {
	home := t.TempDir()
	day := filepath.Join(home, "sessions", "2026", "11", "20")
	data, err := os.ReadFile("testdata/transcript/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(day, "rollout-2026-11-20T14-00-00-x1.jsonl"), data, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(day, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(day, 0o755) })

	got, err := Source{Home: home}.Transcript(session.Session{Agent: "codex", ID: "x1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Skipped != 1 {
		t.Fatalf("got %d messages, %d skipped: %+v", len(got.Messages), got.Skipped, got.Messages)
	}
	user, reply := got.Messages[0], got.Messages[1]
	if user.Text != "add a test" || !user.Time.Equal(time.Date(2026, 11, 20, 14, 3, 0, 0, time.UTC)) {
		t.Errorf("user = %+v", user)
	}
	if reply.Text != "Added it." || strings.Join(reply.Tools, "|") != "shell bash -lc go test ./...|apply_patch" {
		t.Errorf("reply = %+v", reply)
	}
	md, _ := transcript.Markdown(got, time.Now())
	for _, hidden := range []string{"DEVELOPER", "INJECTED", "HIDDEN", "TOOL OUTPUT"} {
		if strings.Contains(md, hidden) {
			t.Errorf("export shows %q", hidden)
		}
	}
}

func TestTranscript_NotFound(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := (Source{Home: home}).Transcript(session.Session{ID: "nope"}); err == nil {
		t.Error("want an error for a missing rollout")
	}
}

func TestStartCommand(t *testing.T) {
	if got := (Source{}).StartCommand("/tmp", "hi"); strings.Join(got, " ") != "codex hi" {
		t.Errorf("got %q", got)
	}
}
