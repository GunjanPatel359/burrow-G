package launch

import (
	"slices"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestDetect(t *testing.T) {
	if term := Detect(env(map[string]string{"GNOME_TERMINAL_SCREEN": "/org/gnome/x"})); term == nil || term.Name() != "gnome-terminal" {
		t.Errorf("inside GNOME Terminal: got %v, want gnome-terminal", term)
	}
	if term := Detect(env(map[string]string{})); term != nil {
		t.Errorf("unknown terminal: got %v, want nil", term)
	}
	inGoLand := map[string]string{"GNOME_TERMINAL_SCREEN": "/org/gnome/x", "TERMINAL_EMULATOR": "JetBrains-JediTerm"}
	if term := Detect(env(inGoLand)); term != nil {
		t.Errorf("inside GoLand's terminal: got %v, want nil (tabs would open in GNOME, not GoLand)", term)
	}
}

func TestGnomeArgs(t *testing.T) {
	got := gnomeArgs("/home/u/proj", "Fix auth", []string{"claude", "--resume", "abc"})
	want := []string{"--tab", "--title=Fix auth", "--working-directory=/home/u/proj", "--", "claude", "--resume", "abc"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestGnomeOpenTab_RunsGnomeTerminal(t *testing.T) {
	var gotName string
	var gotArgs []string
	g := gnome{run: func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	}}

	if err := g.OpenTab("/tmp", "T", []string{"codex", "resume", "x"}); err != nil {
		t.Fatal(err)
	}
	if gotName != "gnome-terminal" || !slices.Contains(gotArgs, "--working-directory=/tmp") {
		t.Errorf("ran %q %q", gotName, gotArgs)
	}
}
