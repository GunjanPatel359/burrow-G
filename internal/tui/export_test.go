package tui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/picker"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var eKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}

// exportModel opens the sessions pane on one session of src, exporting into
// a temp folder.
func exportModel(t *testing.T, src fakeSource) model {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	src.list = []session.Session{{Agent: src.name, ID: "aaa", Title: "From files", Dir: "/tmp", LastUsed: time.Now()}}
	m := newModel(store, []session.Source{src}, "/home/u", nil, theme.Env{}, alwaysDark)
	m.exports = filepath.Join(t.TempDir(), "exports")
	m.version = "9.9.9"
	m = deliver(m, m.Init())
	m, _ = press(m, enter)
	return m
}

// saveToFile opens the dialog and chooses Save to file; it returns the
// command that runs the export.
func saveToFile(m model) (model, tea.Cmd) {
	m, _ = press(m, eKey)
	m, cmd := press(m, enter)
	for _, msg := range run(cmd) {
		if c, ok := msg.(picker.ChoseMsg); ok {
			next, more := m.Update(c)
			return next.(model), more
		}
	}
	return m, nil
}

func writesOne(s session.Session, w *portable.Writer) error {
	if err := w.Header(portable.Header{Agent: s.Agent, ID: s.ID}); err != nil {
		return err
	}
	return w.Record(portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "hi"}}})
}

func TestExportKeyWritesFile(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})

	m, cmd := saveToFile(m)
	if !strings.Contains(m.status.View(), "exporting") {
		t.Errorf("status while exporting = %q", m.status.View())
	}
	m = deliver(m, cmd)

	path := filepath.Join(m.exports, "claude-aaa.jsonl")
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file: %v, mode %v; want mode 600", err, info.Mode().Perm())
	}
	if info, _ := os.Stat(m.exports); info.Mode().Perm() != 0o700 {
		t.Errorf("folder mode = %v, want 700", info.Mode().Perm())
	}
	f, _ := os.Open(path)
	defer f.Close()
	r, err := portable.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.Header().Burrow != "9.9.9" {
		t.Errorf("header burrow = %q, want the app's version", r.Header().Burrow)
	}
	if want := `exported "From files"`; !strings.Contains(m.status.View(), want) {
		t.Errorf("status = %q, want %q", m.status.View(), want)
	}
}

// A done message clears itself; tests make the wait 1 ms (TestMain).
func TestExportedMessageFades(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})
	m, cmd := saveToFile(m)

	var fade tea.Cmd
	for _, msg := range run(cmd) {
		if done, ok := msg.(exportedMsg); ok {
			var next tea.Model
			next, fade = m.Update(done)
			m = next.(model)
		}
	}
	if fade == nil {
		t.Fatal("no fade command after the export finished")
	}
	next, _ := m.Update(fade())
	if strings.Contains(next.(model).status.View(), "exported") {
		t.Errorf("status still shows the message: %q", next.(model).status.View())
	}
}

func TestExportKeyNarrowsExistingFile(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})
	path := filepath.Join(m.exports, "claude-aaa.jsonl")
	os.MkdirAll(m.exports, 0o755)
	if err := os.WriteFile(path, []byte("an older, longer export\n\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, cmd := saveToFile(m)
	m = deliver(m, cmd)

	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v (%v), want 600", info.Mode().Perm(), err)
	}
	if data, _ := os.ReadFile(path); strings.Contains(string(data), "older") {
		t.Error("old content is still in the file")
	}
}

func TestExportKeyUnsupported(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude"})

	m, cmd := saveToFile(m)
	m = deliver(m, cmd)

	if want := "error: export isn't supported for Claude Code yet"; !strings.Contains(m.status.View(), want) {
		t.Errorf("status = %q, want %q", m.status.View(), want)
	}
	if _, err := os.Stat(filepath.Join(m.exports, "claude-aaa.jsonl")); !os.IsNotExist(err) {
		t.Errorf("a file was left behind: %v", err)
	}
}

func TestExportKeyFailureRemovesFile(t *testing.T) {
	fails := func(s session.Session, w *portable.Writer) error {
		w.Header(portable.Header{Agent: s.Agent, ID: s.ID})
		return errors.New("disk full")
	}
	m := exportModel(t, fakeSource{name: "claude", export: fails})

	m, cmd := saveToFile(m)
	m = deliver(m, cmd)

	if !strings.Contains(m.status.View(), "error: ") || !strings.Contains(m.status.View(), "disk full") {
		t.Errorf("status = %q", m.status.View())
	}
	if _, err := os.Stat(filepath.Join(m.exports, "claude-aaa.jsonl")); !os.IsNotExist(err) {
		t.Errorf("half an export was left behind: %v", err)
	}
}

func TestExportKeyTypedWhileSearching(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})

	m, cmd := saveToFile(m)
	m = deliver(m, cmd)

	if _, err := os.Stat(m.exports); !os.IsNotExist(err) {
		t.Error("e while searching exported instead of typing into the search box")
	}
}

func TestExportKeyHint(t *testing.T) {
	if lines := keyLines(false, theme.Unicode); !slices.Contains(lines, "e export") {
		t.Errorf("key list %q has no %q", lines, "e export")
	}
}

// A failed re-export must not take the earlier good export with it.
func TestExportFailureKeepsEarlierExport(t *testing.T) {
	failing := false
	src := fakeSource{name: "claude", export: func(s session.Session, w *portable.Writer) error {
		if failing {
			w.Header(portable.Header{Agent: s.Agent, ID: s.ID})
			return errors.New("disk full")
		}
		return writesOne(s, w)
	}}
	m := exportModel(t, src)
	m, cmd := press(m, eKey)
	m = deliver(m, cmd)

	failing = true
	m, cmd = press(m, eKey)
	m = deliver(m, cmd)

	f, err := os.Open(filepath.Join(m.exports, "claude-aaa.jsonl"))
	if err != nil {
		t.Fatalf("the earlier export is gone: %v", err)
	}
	defer f.Close()
	r, err := portable.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := r.Next(); err != nil {
			if !errors.Is(err, io.EOF) {
				t.Errorf("the earlier export was damaged: %v", err)
			}
			break
		}
	}
	if left, _ := os.ReadDir(m.exports); len(left) != 1 {
		t.Errorf("exports folder holds %d files, want only the export", len(left))
	}
}

// An id from an agent's data must not lead the export out of its folder.
func TestExportRefusesIDThatLeavesFolder(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a", "b", "exports")
	src := fakeSource{name: "codex", export: writesOne}
	for _, id := range []string{"x/../../../victim", `x\y`, "..", "."} {
		msg := export(src, session.Session{Agent: "codex", ID: id}, dir, "v")().(exportedMsg)
		if msg.err == nil {
			t.Errorf("id %q was exported", id)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "victim.jsonl")); !os.IsNotExist(err) {
		t.Error("a file was written outside the exports folder")
	}
}
