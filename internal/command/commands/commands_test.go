package commands

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func testEnv(t *testing.T) command.Env {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return command.Env{
		Store:  store,
		Themes: []string{"dracula", "nord"},
		Current: func() string {
			v, _ := store.Setting("theme")
			return v
		},
	}
}

func TestTheme_SavesAndReturnsEvent(t *testing.T) {
	env := testEnv(t)
	ev, err := New(env).Run("theme nord")
	if err != nil {
		t.Fatal(err)
	}
	if ev != (event.ThemeChanged{Name: "nord"}) {
		t.Fatalf("event = %#v", ev)
	}
	if got, _ := env.Store.Setting("theme"); got != "nord" {
		t.Fatalf("saved theme = %q", got)
	}
}

func TestRefreshAndQuit(t *testing.T) {
	s := New(testEnv(t))
	if ev, err := s.Run("refresh"); err != nil || ev != (event.Refresh{}) {
		t.Fatalf("refresh = %#v, %v", ev, err)
	}
	if ev, err := s.Run("quit"); err != nil || ev != (event.Quit{}) {
		t.Fatalf("quit = %#v, %v", ev, err)
	}
}

func TestHelp_ListsKeysAndCommands(t *testing.T) {
	env := testEnv(t)
	env.Keys = []string{"↑/↓ move"}
	ev, err := New(env).Run("help")
	if err != nil {
		t.Fatal(err)
	}
	text := ev.(event.ShowHelp).Text
	for _, want := range []string{"Keys", "↑/↓ move", "Commands", "theme <value>", "refresh", "quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("help text missing %q:\n%s", want, text)
		}
	}
}

func TestNew_SuggestsAllCommands(t *testing.T) {
	rows := New(testEnv(t)).Suggest("")
	var names []string
	keys := map[string]string{}
	for _, r := range rows {
		names = append(names, r.Text)
		keys[r.Text] = r.Key
	}
	if want := []string{"theme", "refresh", "export", "handoff", "help", "quit"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v", names)
	}
	if keys["refresh"] != "r" || keys["export"] != "e" || keys["handoff"] != "t" {
		t.Fatalf("refresh key = %q", keys["refresh"])
	}
}

func TestTheme_SuggestMarksSavedAndPreviews(t *testing.T) {
	env := command.Env{Themes: []string{"dracula", "nord"}, Current: func() string { return "nord" }}
	rows := suggestTheme(env, "")
	if rows[1].Text != "nord" || rows[1].Hint != "✓" {
		t.Errorf("rows = %+v", rows)
	}
	if p, ok := rows[0].Preview.(event.ThemeChanged); !ok || p.Name != "dracula" || !p.Preview {
		t.Errorf("preview = %#v", rows[0].Preview)
	}
}

func TestTheme_UnknownName(t *testing.T) {
	if _, err := New(testEnv(t)).Run("theme pink"); err == nil || !strings.Contains(err.Error(), `no theme "pink"`) {
		t.Errorf("err = %v", err)
	}
}

type fakeAgent struct{ name, bin string }

func (f fakeAgent) Name() string                           { return f.name }
func (f fakeAgent) List() ([]session.Session, error)       { return nil, nil }
func (f fakeAgent) ResumeCommand(session.Session) []string { return nil }
func (f fakeAgent) Delete(session.Session, string) error   { return nil }
func (f fakeAgent) Transcript(session.Session) (session.Transcript, error) {
	return session.Transcript{}, nil
}
func (f fakeAgent) StartCommand(dir, prompt string) []string {
	if f.bin == "" {
		return nil
	}
	return []string{f.bin, prompt}
}

func transferEnv(t *testing.T, sel *session.Session) command.Env {
	env := testEnv(t)
	env.Sources = []session.Source{fakeAgent{"claude", "claude"}, fakeAgent{"qoder", ""}, fakeAgent{"codex", "codex"}}
	env.LookPath = func(bin string) (string, error) {
		if bin == "codex" {
			return "", errors.New("not found")
		}
		return "/bin/" + bin, nil
	}
	env.Selected = func() (session.Session, bool) {
		if sel == nil {
			return session.Session{}, false
		}
		return *sel, true
	}
	return env
}

func TestExportAndHandoff_Events(t *testing.T) {
	s := session.Session{Agent: "codex", ID: "x1"}
	svc := New(transferEnv(t, &s))

	if ev, err := svc.Run("export"); err != nil || ev != (event.ExportRequested{Session: s}) {
		t.Errorf("export = %#v, %v", ev, err)
	}
	if ev, err := svc.Run("handoff"); err != nil || ev != (event.HandoffRequested{Session: s}) {
		t.Errorf("handoff = %#v, %v", ev, err)
	}
	if ev, err := svc.Run("handoff @agent:claude"); err != nil || ev != (event.HandoffRequested{Session: s, Agent: "claude"}) {
		t.Errorf("handoff @agent:claude = %#v, %v", ev, err)
	}
}

func TestExportAndHandoff_Refusals(t *testing.T) {
	s := session.Session{Agent: "codex", ID: "x1"}
	tests := []struct {
		line string
		sel  *session.Session
		want string
	}{
		{"export", nil, "select a session first"},
		{"handoff", nil, "select a session first"},
		{"handoff @agent:codex", &s, `no installed agent "codex" (choose from: claude)`},
		{"handoff @agent:qoder", &s, `no installed agent "qoder" (choose from: claude)`},
	}
	for _, tt := range tests {
		_, err := New(transferEnv(t, tt.sel)).Run(tt.line)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", tt.line, err, tt.want)
		}
	}
}

func TestHandoff_SuggestsInstalledTargets(t *testing.T) {
	rows := suggestTarget(transferEnv(t, nil), "")
	if len(rows) != 1 || rows[0].Text != "claude" {
		t.Errorf("rows = %+v", rows)
	}
}
