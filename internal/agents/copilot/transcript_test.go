package copilot

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Transcriber = Source{}

func lockDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
}

func TestTranscript_Events(t *testing.T) {
	home := t.TempDir()
	events := filepath.Join(home, "session-state", "c1", "events.jsonl")
	data, err := os.ReadFile("testdata/transcript/events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(events), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(events, data, 0o400); err != nil {
		t.Fatal(err)
	}
	lockDir(t, filepath.Dir(events))

	got, err := Source{DBPath: filepath.Join(home, "session-store.db")}.Transcript(session.Session{Agent: "copilot", ID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Skipped != 1 {
		t.Fatalf("got %d messages, %d skipped: %+v", len(got.Messages), got.Skipped, got.Messages)
	}
	if got.Messages[0].Text != "review the app" {
		t.Errorf("user = %+v", got.Messages[0])
	}
	if reply := got.Messages[1]; reply.Text != "Looks good." || strings.Join(reply.Tools, "|") != "view app.go" {
		t.Errorf("reply = %+v", reply)
	}
}

func TestTranscript_TurnsFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-store.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE turns (id INTEGER, session_id TEXT, turn_index INTEGER, user_message TEXT, assistant_response TEXT, timestamp TEXT);
		INSERT INTO turns VALUES (2, 'c1', 1, 'and now?', 'Done.', '2026-11-20T14:05:00Z');
		INSERT INTO turns VALUES (1, 'c1', 0, 'start', NULL, '2026-11-20T14:03:00Z');
		INSERT INTO turns VALUES (3, 'c2', 0, 'OTHER', 'x', '2026-11-20T14:03:00Z');
	`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	lockDir(t, filepath.Dir(path))

	got, err := Source{DBPath: path}.Transcript(session.Session{Agent: "copilot", ID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	var roles []string
	for _, m := range got.Messages {
		roles = append(roles, m.Role+":"+m.Text)
	}
	// The empty first reply is dropped, so the two prompts merge.
	if got := strings.Join(roles, "|"); got != "user:start\n\nand now?|assistant:Done." {
		t.Errorf("messages = %q", got)
	}
}

func TestStartCommand(t *testing.T) {
	if got := (Source{}).StartCommand("/tmp", "hi"); strings.Join(got, " ") != "copilot -i hi" {
		t.Errorf("got %q", got)
	}
}
