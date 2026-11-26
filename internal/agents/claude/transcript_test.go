package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

var _ session.Transcriber = Source{}

// readOnlyHome copies fixture into a home whose files can't be written,
// so reading proves it never needs write access.
func readOnlyHome(t *testing.T, fixture, id string) string {
	t.Helper()
	home := t.TempDir()
	proj := filepath.Join(home, "projects", "-tmp-proj")
	copyFile(t, fixture, filepath.Join(proj, id+".jsonl"))
	if err := os.Chmod(filepath.Join(proj, id+".jsonl"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(proj, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(proj, 0o755) })
	return home
}

func TestTranscript(t *testing.T) {
	home := readOnlyHome(t, "testdata/transcript/main.jsonl", "aaa")
	s := session.Session{Agent: "claude", ID: "aaa", Title: "Tests", Dir: "/tmp/proj"}

	got, err := Source{Home: home}.Transcript(s)
	if err != nil {
		t.Fatal(err)
	}
	at := func(sec int) time.Time { return time.Date(2026, 11, 20, 14, 3, sec, 0, time.UTC) }
	want := []session.Message{
		{Role: "user", Time: at(0), Text: "make the tests pass"},
		{Role: "assistant", Time: at(6), Text: "Running them now.", Tools: []string{"Bash go test ./..."}},
		{Role: "assistant", Time: at(12), Text: "All green."},
		{Role: "user", Time: time.Date(2026, 11, 20, 14, 5, 0, 0, time.UTC), Text: "/commit"},
	}
	if len(got.Messages) != len(want) {
		t.Fatalf("got %d messages, want %d: %+v", len(got.Messages), len(want), got.Messages)
	}
	for i := range want {
		g, w := got.Messages[i], want[i]
		if g.Role != w.Role || !g.Time.Equal(w.Time) || g.Text != w.Text || strings.Join(g.Tools, "|") != strings.Join(w.Tools, "|") {
			t.Errorf("message %d = %+v\nwant %+v", i, g, w)
		}
	}
	if got.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", got.Skipped)
	}

	md, _ := transcript.Markdown(got, time.Now())
	for _, hidden := range []string{"SECRET THOUGHTS", "TOOL OUTPUT", "SIDECHAIN", "META CAVEAT", "noise"} {
		if strings.Contains(md, hidden) {
			t.Errorf("export shows %q", hidden)
		}
	}
	if !strings.Contains(md, "(1 entries skipped)") {
		t.Error("export misses the skipped count")
	}
}

func TestTranscript_AllCorrupt(t *testing.T) {
	home := readOnlyHome(t, "testdata/transcript/corrupt.jsonl", "bad")
	got, err := Source{Home: home}.Transcript(session.Session{Agent: "claude", ID: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Skipped != 2 || len(got.Messages) != 0 {
		t.Errorf("got %d messages, %d skipped", len(got.Messages), got.Skipped)
	}
	if err := transcript.CheckMessages(got); err == nil || err.Error() != "no messages found in claude session bad" {
		t.Errorf("err = %v", err)
	}
}

func TestTranscript_NotFound(t *testing.T) {
	if _, err := (Source{Home: t.TempDir()}).Transcript(session.Session{ID: "nope"}); err == nil {
		t.Error("want an error for a missing session file")
	}
}

func TestStartCommand(t *testing.T) {
	if got := (Source{}).StartCommand("/tmp", "hi"); strings.Join(got, " ") != "claude hi" {
		t.Errorf("got %q", got)
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
