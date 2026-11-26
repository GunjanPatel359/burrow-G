package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/agents/claude"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

const claudeSession = `{"type":"user","timestamp":"2026-11-20T14:03:00Z","cwd":"%DIR%","message":{"content":"ship it with AKIAABCDEFGHIJKLMNOP"}}
{"type":"assistant","timestamp":"2026-11-20T14:03:05Z","message":{"content":[{"type":"text","text":"Shipped."}]}}
`

// fakeTarget is an installed agent that takes a first prompt.
type fakeTarget struct{ fakeSource }

func (fakeTarget) Transcript(session.Session) (session.Transcript, error) {
	return session.Transcript{}, nil
}
func (f fakeTarget) StartCommand(dir, prompt string) []string { return []string{f.name, prompt} }

func transferSetup(t *testing.T) (*db.DB, []session.Source, string) {
	t.Helper()
	home := t.TempDir()
	dir := t.TempDir()
	file := filepath.Join(home, "projects", "-p", "abc123.jsonl")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(strings.ReplaceAll(claudeSession, "%DIR%", dir)), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), version)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	sources := []session.Source{claude.Source{Home: home}, fakeTarget{fakeSource{name: "codex"}}, fakeSource{name: "old"}}
	refreshAll(store, sources, &bytes.Buffer{})
	return store, sources, dir
}

func TestParseExportArgs(t *testing.T) {
	tests := []struct {
		args          []string
		id, out, fail string
	}{
		{args: []string{"abc"}, id: "abc"},
		{args: []string{"abc", "-o", "f.md"}, id: "abc", out: "f.md"},
		{args: []string{"-o", "f.md", "abc"}, id: "abc", out: "f.md"},
		{args: []string{"abc", "-o"}, fail: "-o needs a file name"},
		{args: nil, fail: "usage: burrow-g export <id> [-o file]"},
		{args: []string{"a", "b"}, fail: `unexpected argument "b"`},
	}
	for _, tt := range tests {
		id, out, err := parseExportArgs(tt.args)
		if tt.fail != "" {
			if err == nil || err.Error() != tt.fail {
				t.Errorf("%q: err = %v, want %q", tt.args, err, tt.fail)
			}
			continue
		}
		if err != nil || id != tt.id || out != tt.out {
			t.Errorf("%q: got %q %q %v", tt.args, id, out, err)
		}
	}
}

func TestRunExport(t *testing.T) {
	store, sources, dir := transferSetup(t)
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

	var stdout, stderr bytes.Buffer
	if err := runExport(store, sources, "abc", "", &stdout, &stderr, now); err != nil {
		t.Fatal(err)
	}
	md := stdout.String()
	if !strings.Contains(md, "Agent: claude · Session: abc123 · Folder: "+dir) || !strings.Contains(md, "Shipped.") {
		t.Errorf("markdown:\n%s", md)
	}
	if strings.Contains(md, "AKIA") || !strings.Contains(stderr.String(), "2 secrets masked") {
		t.Errorf("secret not masked or not reported: %s", stderr.String())
	}

	out := filepath.Join(t.TempDir(), "s.md")
	if err := os.WriteFile(out, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := runExport(store, sources, "abc", out, &stdout, &stderr, now); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), out+"\n") {
		t.Errorf("mode %v, stdout %q, stderr %q", info.Mode().Perm(), stdout.String(), stderr.String())
	}
}

func TestPrepareHandoff(t *testing.T) {
	store, sources, dir := transferSetup(t)
	found := func(string) (string, error) { return "/bin/x", nil }

	s, argv, st, err := prepareHandoff(store, sources, "abc", "codex", found)
	if err != nil {
		t.Fatal(err)
	}
	if s.Dir != dir || len(argv) != 2 || argv[0] != "codex" || !strings.HasPrefix(argv[1], "You are continuing a coding session that started in claude") {
		t.Errorf("session %+v argv %.120q", s, argv)
	}
	if st.Messages != 2 || st.Masked != 3 { // title in preamble and header, plus the message
		t.Errorf("stats = %+v", st)
	}
}

func TestPrepareHandoff_Refusals(t *testing.T) {
	store, sources, _ := transferSetup(t)
	found := func(string) (string, error) { return "/bin/x", nil }
	missing := func(string) (string, error) { return "", errors.New("no") }

	tests := []struct {
		agent    string
		lookPath func(string) (string, error)
		want     string
	}{
		{"nope", found, `unknown agent "nope" (choose from: claude, codex)`},
		{"old", found, "old can't start with a first prompt yet"},
		{"codex", missing, "codex is not installed (not found on PATH)"},
	}
	for _, tt := range tests {
		_, argv, _, err := prepareHandoff(store, sources, "abc", tt.agent, tt.lookPath)
		if err == nil || err.Error() != tt.want || argv != nil {
			t.Errorf("%s: err = %v, argv = %q, want %q", tt.agent, err, argv, tt.want)
		}
	}
}
