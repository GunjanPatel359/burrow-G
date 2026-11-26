package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var keyO = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}

// handoffModel opens the sessions pane on one Claude session in a real temp
// folder, with only the agents in found on PATH.
func handoffModel(t *testing.T, term launch.Terminal, src fakeSource, found ...string) (model, string) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	dir := t.TempDir()
	src.list = []session.Session{{Agent: src.name, ID: "aaa", Title: "From files", Dir: dir, LastUsed: time.Now()}}
	sources := []session.Source{src, fakeSource{name: "qoder"}, fakeSource{name: "codex"}}
	m := newModel(store, sources, "/home/u", term, theme.Env{}, alwaysDark)
	data := t.TempDir()
	m.exports, m.handoffs = filepath.Join(data, "exports"), filepath.Join(data, "handoffs")
	m.lookPath = func(name string) (string, error) {
		if slices.Contains(found, name) {
			return "/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
	m = deliver(m, m.Init())
	m, _ = press(m, enter)
	return m, dir
}

func claudeSource() fakeSource { return fakeSource{name: "claude", export: writesOne} }

// choose opens the dialog, moves down n rows and presses k, then feeds every
// message back; it returns the model and whether the app quit.
func choose(m model, n int, k tea.KeyMsg) (model, bool) {
	m, _ = press(m, eKey)
	for range n {
		m, _ = press(m, down)
	}
	m, cmd := press(m, k)
	return runAll(m, cmd)
}

// runAll is deliver that also reports a quit.
func runAll(m model, cmd tea.Cmd) (model, bool) {
	quit := false
	for _, msg := range run(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			quit = true
			continue
		}
		if !ours(msg) { // spinner and fade ticks would tick forever
			continue
		}
		next, more := m.Update(msg)
		var q bool
		m, q = runAll(next.(model), more)
		quit = quit || q
	}
	return m, quit
}

func TestEOpensDialog(t *testing.T) {
	m, _ := handoffModel(t, nil, claudeSource(), "claude", "codex")
	m, _ = press(m, eKey)
	v := m.View()
	for _, want := range []string{"Move session", "Save to file", "Continue in Claude Code", "Continue in Codex"} {
		if !strings.Contains(v, want) {
			t.Errorf("dialog is missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "Continue in Qoder") {
		t.Error("Qoder isn't on PATH but is offered")
	}
}

func TestDialogSaveToFile(t *testing.T) {
	m, _ := handoffModel(t, nil, claudeSource(), "codex")
	m, _ = choose(m, 0, enter)
	if _, err := os.Stat(filepath.Join(m.exports, "claude-aaa.jsonl")); err != nil {
		t.Errorf("no export: %v", err)
	}
	if !strings.Contains(m.status.View(), `exported "From files"`) {
		t.Errorf("status = %q", m.status.View())
	}
}

func TestContinueHere(t *testing.T) {
	m, dir := handoffModel(t, nil, claudeSource(), "claude", "codex")
	m, quit := choose(m, 2, enter) // Save to file, Claude Code, Codex

	path := filepath.Join(m.handoffs, "claude-aaa.md")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("handoff file: %v", err)
	}
	if data, _ := os.ReadFile(path); !strings.HasPrefix(string(data), "# Session handed off from Claude Code") {
		t.Errorf("handoff file starts %q", string(data[:min(len(data), 60)]))
	}
	l := m.launch
	if l == nil || l.Dir != dir || l.Argv[0] != "codex" || !strings.Contains(l.Argv[1], path) || !quit {
		t.Errorf("launch = %+v, quit %v; want codex with a prompt naming %s", l, quit, path)
	}
}

func TestContinueNewTab(t *testing.T) {
	term := &fakeTerminal{}
	m, dir := handoffModel(t, term, claudeSource(), "claude", "codex")
	m, quit := choose(m, 2, enter)

	if len(term.opened) != 1 || !strings.HasPrefix(term.opened[0], dir+"|From files|codex Read ") || quit || m.launch != nil {
		t.Errorf("opened %q, quit %v, launch %+v", term.opened, quit, m.launch)
	}
	if !strings.Contains(m.status.View(), `continued "From files" in Codex`) {
		t.Errorf("status = %q", m.status.View())
	}
}

func TestContinueHereWithO(t *testing.T) {
	term := &fakeTerminal{}
	m, _ := handoffModel(t, term, claudeSource(), "claude", "codex")
	m, quit := choose(m, 2, keyO)
	if len(term.opened) != 0 || m.launch == nil || !quit {
		t.Errorf("o should continue here: opened %q, launch %+v", term.opened, m.launch)
	}
}

func TestContinueMissingFolder(t *testing.T) {
	term := &fakeTerminal{}
	m, dir := handoffModel(t, term, claudeSource(), "codex")
	os.RemoveAll(dir)
	m, quit := choose(m, 1, enter)

	if !strings.Contains(m.status.View(), "error: ") || len(term.opened) != 0 || m.launch != nil || quit {
		t.Errorf("status %q, opened %q, launch %+v", m.status.View(), term.opened, m.launch)
	}
	if _, err := os.Stat(filepath.Join(m.handoffs, "claude-aaa.md")); !os.IsNotExist(err) {
		t.Error("a handoff file was written for a session whose folder is gone")
	}
}

func TestContinueUnsupportedExport(t *testing.T) {
	m, _ := handoffModel(t, nil, fakeSource{name: "claude"}, "codex")
	m, quit := choose(m, 1, enter)
	if want := "error: export isn't supported for Claude Code yet"; !strings.Contains(m.status.View(), want) || m.launch != nil || quit {
		t.Errorf("status %q, launch %+v", m.status.View(), m.launch)
	}
	if _, err := os.Stat(filepath.Join(m.handoffs, "claude-aaa.md")); !os.IsNotExist(err) {
		t.Error("a handoff file was written without an export")
	}
}

func TestNoAgentsOnPath(t *testing.T) {
	m, _ := handoffModel(t, nil, claudeSource())
	m, _ = press(m, eKey)
	if v := m.View(); !strings.Contains(v, "Save to file") || strings.Contains(v, "Continue in") {
		t.Errorf("with no agent on PATH only Save to file shows:\n%s", v)
	}
}

func TestDialogEscDoesNothing(t *testing.T) {
	m, _ := handoffModel(t, nil, claudeSource(), "codex")
	m, quit := choose(m, 1, esc)
	if m.picker.Opened() || strings.Contains(m.View(), "Move session") || m.launch != nil || quit {
		t.Errorf("esc should close the dialog and do nothing:\n%s", m.View())
	}
	if _, err := os.Stat(m.exports); !os.IsNotExist(err) {
		t.Error("esc exported anyway")
	}
}

func TestDialogTakesKeys(t *testing.T) {
	m, _ := handoffModel(t, nil, claudeSource(), "codex")
	m, _ = press(m, eKey)
	m, cmd := press(m, keyQ)
	if quits(cmd) || !m.picker.Opened() {
		t.Error("q inside the dialog quit the app")
	}
}
