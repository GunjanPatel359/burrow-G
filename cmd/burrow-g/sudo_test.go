package main

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestHomeDir(t *testing.T) {
	root := func() (*user.User, error) { return &user.User{HomeDir: "/root"}, nil }
	envHome := func() (string, error) { return "/home/u", nil }

	if got, _ := homeDir("", root, envHome); got != "/home/u" {
		t.Errorf("normal run: home = %q, want /home/u", got)
	}
	if got, _ := homeDir("u", root, envHome); got != "/root" {
		t.Errorf("under sudo: home = %q, want /root (HOME may still be the caller's)", got)
	}
	broken := func() (*user.User, error) { return nil, errors.New("no passwd entry") }
	if got, _ := homeDir("u", broken, envHome); got != "/home/u" {
		t.Errorf("under sudo, lookup fails: home = %q, want HOME fallback", got)
	}
}

func TestSudoHint(t *testing.T) {
	notFound := &exec.Error{Name: "claude", Err: exec.ErrNotFound}

	if err := sudoHint(notFound, "u"); !strings.Contains(err.Error(), "burrow-g setup-sudo") {
		t.Errorf("under sudo: err = %v, want a PATH hint", err)
	}
	if err := sudoHint(notFound, ""); err != notFound {
		t.Errorf("normal run: err = %v, want it unchanged", err)
	}
	other := errors.New("folder no longer exists")
	if err := sudoHint(other, "u"); err != other {
		t.Errorf("other errors: err = %v, want it unchanged", err)
	}
}

// The hint must survive the error Resume really returns, not only a
// hand-built *exec.Error.
func TestSudoHint_MissingCLIFromResume(t *testing.T) {
	err := session.Resume(session.Session{Dir: t.TempDir()}, []string{"no-such-agent-cli-xyz"})
	if err := sudoHint(err, "u"); err == nil || !strings.Contains(err.Error(), "burrow-g setup-sudo") {
		t.Errorf("under sudo: err = %v, want a PATH hint", err)
	}
}

func TestSudoWrapper(t *testing.T) {
	got := sudoWrapper("/home/u/bin/burrow-g", "/home/u/.nvm/bin:/usr/bin")
	want := `#!/bin/sh
# Written by 'burrow-g setup-sudo'. Run it again if burrow-G moves.
PATH="$PATH":'/home/u/.nvm/bin:/usr/bin'
exec '/home/u/bin/burrow-g' "$@"
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote(`/home/it's me`); got != `'/home/it'\''s me'` {
		t.Errorf("got %s", got)
	}
}

// fakeSudo records the command and the wrapper it would install.
func fakeSudo(name *string, args *[]string, script *string) func(string, ...string) error {
	return func(n string, a ...string) error {
		*name, *args = n, a
		data, err := os.ReadFile(a[len(a)-1])
		*script = string(data)
		return err
	}
}

func TestSetupSudo_InstallsWrapper(t *testing.T) {
	target := filepath.Join(t.TempDir(), "bin", "burrow-g")
	var name, script string
	var args []string

	if err := setupSudo("/home/u/burrow-g", "/home/u/bin", target, "", fakeSudo(&name, &args, &script)); err != nil {
		t.Fatal(err)
	}
	if name != "sudo" || !slices.Contains(args, target) {
		t.Errorf("ran %s %q", name, args)
	}
	if script != sudoWrapper("/home/u/burrow-g", "/home/u/bin") {
		t.Errorf("installed:\n%s", script)
	}
}

func TestSetupSudo_ReplacesItsOwnWrapper(t *testing.T) {
	target := filepath.Join(t.TempDir(), "burrow-g")
	if err := os.WriteFile(target, []byte(sudoWrapper("/old/burrow-g", "/old")), 0o755); err != nil {
		t.Fatal(err)
	}
	var name, script string
	var args []string

	if err := setupSudo("/new/burrow-g", "/new", target, "", fakeSudo(&name, &args, &script)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "/new/burrow-g") {
		t.Errorf("installed:\n%s", script)
	}
}

func TestSetupSudo_Refuses(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("#!/bin/sh\necho someone else's\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	never := func(string, ...string) error { t.Fatal("should not run sudo"); return nil }

	cases := map[string]struct{ binary, target, sudoUser string }{
		"run under sudo":            {"/home/u/burrow-g", filepath.Join(dir, "x"), "u"},
		"already in the system dir": {filepath.Join(dir, "x"), filepath.Join(dir, "x"), ""},
		"someone else's file":       {"/home/u/burrow-g", other, ""},
	}
	for name, c := range cases {
		if err := setupSudo(c.binary, "/p", c.target, c.sudoUser, never); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
