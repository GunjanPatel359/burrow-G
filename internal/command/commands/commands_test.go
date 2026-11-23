package commands

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
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
	if want := []string{"theme", "refresh", "help", "quit"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v", names)
	}
	if keys["refresh"] != "r" {
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
