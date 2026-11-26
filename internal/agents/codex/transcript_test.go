package codex

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
	if len(got.Messages) != 3 || got.Skipped != 1 {
		t.Fatalf("got %d messages, %d skipped: %+v", len(got.Messages), got.Skipped, got.Messages)
	}
	user, tools, reply := got.Messages[0], got.Messages[1], got.Messages[2]
	if user.Text != "add a test" || !user.Time.Equal(time.Date(2026, 11, 20, 14, 3, 0, 0, time.UTC)) {
		t.Errorf("user = %+v", user)
	}
	if strings.Join(tools.Tools, "|") != "shell bash -lc go test ./...|apply_patch" || reply.Text != "Added it." {
		t.Errorf("tools = %+v, reply = %+v", tools, reply)
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

func TestCustomInput(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"one command", `text(await tools.exec_command({cmd:"go test ./..."}))`, "exec go test ./..."},
		{"escaped quotes", `text(await tools.exec_command({cmd:"rg \"a b\" docs"}))`, `exec rg "a b" docs`},
		{"single quotes", `text(await tools.exec_command({cmd:'sed -n \'1,5p\' a.go'}))`, "exec sed -n '1,5p' a.go"},
		{"several commands", "await Promise.allSettled([\ntools.exec_command({cmd:\"ls\"}),\ntools.exec_command({cmd:\"pwd\"})])", "exec ls (+1 more)"},
		{"inner tool", `text(await tools.apply_patch("*** Begin Patch"))`, "exec apply_patch"},
		{"plain text", `*** Begin Patch`, "exec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, _ := json.Marshal(tt.input)
			if got := transcript.ToolLine("exec", customInput(raw)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsInjected(t *testing.T) {
	for in, want := range map[string]bool{
		"<environment_context>\n<cwd>/tmp</cwd>": true,
		"# AGENTS.md instructions for /tmp":      true,
		"<skill>\n<name>develop</name>":          true,
		"<codex_internal_context>x":              true,
		"<div> is misaligned on mobile":          false,
		"add a test":                             false,
	} {
		if got := isInjected(in); got != want {
			t.Errorf("isInjected(%q) = %v, want %v", in, got, want)
		}
	}
}
