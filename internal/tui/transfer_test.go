package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// talker is a fake agent that can export and take a first prompt.
type talker struct {
	fakeSource
	msgs []session.Message
}

func (f talker) Transcript(s session.Session) (session.Transcript, error) {
	return session.Transcript{Session: s, Messages: f.msgs}, nil
}
func (f talker) StartCommand(dir, prompt string) []string { return []string{f.name, prompt} }

var (
	eKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	tKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}
	oKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
)

// transferModel has a claude session in dir and claude plus codex installed.
func transferModel(t *testing.T, term launch.Terminal, dir string) (model, string) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	home := t.TempDir()
	msgs := []session.Message{{Role: "user", Text: "key AKIAABCDEFGHIJKLMNOP"}, {Role: "assistant", Text: "ok"}}
	claude := talker{fakeSource{name: "claude", list: []session.Session{
		{Agent: "claude", ID: "aaa", Title: "From files", Dir: dir, LastUsed: time.Now()},
	}}, msgs}
	codex := talker{fakeSource: fakeSource{name: "codex"}}
	plain := fakeSource{name: "plain"}
	m := newModel(store, []session.Source{claude, codex, plain}, home, term, theme.Env{}, alwaysDark)
	m.lookPath = func(string) (string, error) { return "/bin/x", nil }
	m = deliver(m, m.Init())
	m, _ = press(m, enter) // to the sessions pane
	return m, home
}

func TestExportKeyWritesPrivateFile(t *testing.T) {
	m, home := transferModel(t, nil, t.TempDir())
	m = pressAll(m, eKey)

	path := filepath.Join(home, ".local", "share", "burrow-g", "exports", "claude-aaa.md")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("no export file: %v\n%s", err, m.View())
	}
	dir, _ := os.Stat(filepath.Dir(path))
	if info.Mode().Perm() != 0o600 || dir.Mode().Perm() != 0o700 {
		t.Errorf("file %v, folder %v", info.Mode().Perm(), dir.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "AKIA") {
		t.Error("export left the key in")
	}
	if v := m.View(); !strings.Contains(v, "1 secrets masked") || !strings.Contains(v, "claude-aaa.md") {
		t.Errorf("status should show the path and masked count:\n%s", v)
	}
}

func TestHandoffKeyOpensPickerOfInstalledTargets(t *testing.T) {
	m, _ := transferModel(t, nil, t.TempDir())
	m = pressAll(m, tKey)
	if !m.picker.Opened() {
		t.Fatalf("t should open the picker:\n%s", m.View())
	}
	v := m.picker.View(m.width)
	if !strings.Contains(m.View(), "Hand off session") || !strings.Contains(v, "Codex") || strings.Contains(v, "plain") {
		t.Errorf("picker should list claude and codex only:\n%s", v)
	}

	m = pressAll(m, esc)
	if m.picker.Opened() || m.chosen != nil {
		t.Error("esc should close the picker and launch nothing")
	}
}

func TestHandoffHereQuitsWithArgvAndReport(t *testing.T) {
	m, _ := transferModel(t, nil, t.TempDir())
	m = pressAll(m, tKey)
	m = pressAll(m, down) // codex
	m = pressAll(m, enter)

	if m.chosen == nil {
		t.Fatalf("enter without tabs should hand off here:\n%s", m.View())
	}
	if argv := m.chosen.Argv; len(argv) != 2 || argv[0] != "codex" || !strings.HasPrefix(argv[1], "You are continuing") {
		t.Errorf("argv = %.80q", argv)
	}
	if !strings.HasPrefix(m.chosen.Report, "handed off to codex: 2 messages") {
		t.Errorf("report = %q", m.chosen.Report)
	}
}

func TestHandoffInNewTab(t *testing.T) {
	term := &fakeTerminal{}
	dir := t.TempDir()
	m, _ := transferModel(t, term, dir)
	m = pressAll(m, tKey)
	m = pressAll(m, enter) // claude, in a new tab

	if m.chosen != nil || len(term.opened) != 1 || !strings.HasPrefix(term.opened[0], dir+"|From files|claude You are continuing") {
		t.Fatalf("opened = %.120q, chosen = %v", term.opened, m.chosen)
	}
	if !strings.Contains(m.View(), "handed off to claude") {
		t.Errorf("status should show the report:\n%s", m.View())
	}

	m = pressAll(m, tKey)
	m = pressAll(m, oKey)
	if m.chosen == nil || len(term.opened) != 1 {
		t.Error("o should hand off here even when tabs work")
	}
}

func TestHandoffRefusesMissingFolder(t *testing.T) {
	m, _ := transferModel(t, nil, filepath.Join(t.TempDir(), "gone"))
	m = pressAll(m, tKey)
	m = pressAll(m, enter)
	if m.chosen != nil || !strings.Contains(m.View(), "folder no longer exists") {
		t.Errorf("chosen = %v\n%s", m.chosen, m.View())
	}
}

func TestHandoffNoAgentInstalled(t *testing.T) {
	m, _ := transferModel(t, nil, t.TempDir())
	m.lookPath = func(string) (string, error) { return "", errors.New("no") }
	m = pressAll(m, tKey)
	if m.picker.Opened() || !strings.Contains(m.View(), "no agent CLI found on PATH") {
		t.Errorf("status:\n%s", m.View())
	}
}
