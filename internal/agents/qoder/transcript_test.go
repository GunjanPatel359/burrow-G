package qoder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestStamp(t *testing.T) {
	tests := []struct {
		raw  string
		want time.Time
	}{
		{`"2026-11-20T14:03:00Z"`, time.Date(2026, 11, 20, 14, 3, 0, 0, time.UTC)},
		{`1794027330689`, time.UnixMilli(1794027330689)},
		{`{"odd":true}`, time.Time{}},
		{``, time.Time{}},
	}
	for _, tt := range tests {
		if got := stamp(json.RawMessage(tt.raw)); !got.Equal(tt.want) {
			t.Errorf("stamp(%s) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestUserText(t *testing.T) {
	tests := []struct{ in, want string }{
		{"fix the build", "fix the build"},
		{"<div> is misaligned on mobile", "<div> is misaligned on mobile"},
		{"<pasted_content>log line</pasted_content> why?", "<pasted_content>log line</pasted_content> why?"},
		{"<command-message>develop</command-message>\n<command-name>/develop</command-name>\n<command-args>auth</command-args>", "/develop auth"},
		{"<command-name>/clear</command-name>\n<command-args></command-args>", "/clear"},
		{"<bash-input>git status</bash-input>", "! git status"},
		{"<bash-stdout>On branch main</bash-stdout><bash-stderr></bash-stderr>", ""},
		{"<task-notification>agent finished</task-notification>", ""},
		{"<local-command-stdout>noise</local-command-stdout>", ""},
		{"[Request interrupted by user]", ""},
		{"[Request interrupted by user for tool use]", ""},
		{"<ide_opened_file>The user opened a.go</ide_opened_file>", ""},
		{"<system-reminder>context</system-reminder>\nwhy does this fail?", "why does this fail?"},
		{"<system-reminders> are confusing me", "<system-reminders> are confusing me"},
	}
	for _, tt := range tests {
		if got := userText("user", tt.in); got != tt.want {
			t.Errorf("userText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := userText("assistant", "<b>bold</b>"); got != "<b>bold</b>" {
		t.Errorf("assistant text must pass through, got %q", got)
	}
}

func TestMessages_SkipsCompactSummary(t *testing.T) {
	var rec record
	raw := `{"type":"user","isCompactSummary":true,"message":{"content":"This session is being continued from a previous conversation."}}`
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatal(err)
	}
	if got := messages(rec); len(got) != 0 {
		t.Errorf("a compaction summary is not a turn: %+v", got)
	}
}
