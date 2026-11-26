package tui

import (
	"errors"
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

func writesOne(s session.Session, w *portable.Writer) error {
	if err := w.Header(portable.Header{Agent: s.Agent, ID: s.ID}); err != nil {
		return err
	}
	return w.Record(portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "hi"}}})
}

func TestExportKeyWritesFile(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})

	m, cmd := press(m, eKey)
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
	if want := `exported 1 record to "` + path + `"`; !strings.Contains(m.status.View(), want) {
		t.Errorf("status = %q, want %q", m.status.View(), want)
	}
}

func TestExportKeyNarrowsExistingFile(t *testing.T) {
	m := exportModel(t, fakeSource{name: "claude", export: writesOne})
	path := filepath.Join(m.exports, "claude-aaa.jsonl")
	os.MkdirAll(m.exports, 0o755)
	if err := os.WriteFile(path, []byte("an older, longer export\n\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, cmd := press(m, eKey)
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

	m, cmd := press(m, eKey)
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

	m, cmd := press(m, eKey)
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

	m, cmd := press(m, eKey)
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
