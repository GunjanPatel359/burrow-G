package launch

import (
	"errors"
	"slices"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestDetect(t *testing.T) {
	if term := detect("linux", env(map[string]string{"GNOME_TERMINAL_SCREEN": "/org/gnome/x"})); term == nil || term.Name() != "gnome-terminal" {
		t.Errorf("inside GNOME Terminal: got %v, want gnome-terminal", term)
	}
	if term := detect("linux", env(map[string]string{})); term != nil {
		t.Errorf("unknown terminal: got %v, want nil", term)
	}
	inGoLand := map[string]string{"GNOME_TERMINAL_SCREEN": "/org/gnome/x", "TERMINAL_EMULATOR": "JetBrains-JediTerm"}
	if term := detect("linux", env(inGoLand)); term != nil {
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

func TestDetect_WindowsTerminal(t *testing.T) {
	inWT := map[string]string{"WT_SESSION": "abc-123"}
	if term := detect("windows", env(inWT)); term == nil || term.Name() != "windows-terminal" {
		t.Errorf("inside Windows Terminal: got %v, want windows-terminal", term)
	}
	if term := detect("linux", env(inWT)); term != nil {
		t.Errorf("inside WSL: got %v, want nil (wt.exe can't use Linux paths)", term)
	}
}

func TestWTArgs(t *testing.T) {
	got := wtArgs(`C:\proj`, "Fix a; b", []string{`C:\bin\claude.cmd`, "--resume", "abc"})
	want := []string{"-w", "0", "new-tab", "--title", `Fix a\; b`, "-d", `C:\proj`, `C:\bin\claude.cmd`, "--resume", "abc"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestWTOpenTab_UsesFullAgentPath(t *testing.T) {
	var gotName string
	var gotArgs []string
	w := windowsTerminal{
		run: func(name string, args ...string) error {
			gotName, gotArgs = name, args
			return nil
		},
		lookPath: func(string) (string, error) { return `C:\npm\codex.cmd`, nil },
	}

	if err := w.OpenTab(`C:\proj`, "T", []string{"codex", "resume", "x"}); err != nil {
		t.Fatal(err)
	}
	if gotName != "wt.exe" || !slices.Contains(gotArgs, `C:\npm\codex.cmd`) || slices.Contains(gotArgs, "codex") {
		t.Errorf("ran %q %q", gotName, gotArgs)
	}
}

func TestWTOpenTab_AgentNotInstalled(t *testing.T) {
	w := windowsTerminal{
		run:      func(string, ...string) error { t.Fatal("should not run wt.exe"); return nil },
		lookPath: func(string) (string, error) { return "", errors.New("not found") },
	}
	if err := w.OpenTab(`C:\proj`, "T", []string{"codex"}); err == nil {
		t.Error("want an error when the agent isn't installed")
	}
}
