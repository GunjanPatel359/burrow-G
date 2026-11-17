package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// homeDir is the home of the user we run as. Under sudo it comes from the
// user database, because HOME may still point at the caller's folder.
func homeDir(sudoUser string, current func() (*user.User, error), fallback func() (string, error)) (string, error) {
	if sudoUser != "" {
		if u, err := current(); err == nil && u.HomeDir != "" {
			return u.HomeDir, nil
		}
	}
	return fallback()
}

// sudoHint explains a missing agent under sudo, which drops the caller's PATH.
func sudoHint(err error, sudoUser string) error {
	if sudoUser == "" || !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	return fmt.Errorf("%w (sudo uses its own PATH, run 'burrow-g setup-sudo' once without sudo)", err)
}

// sudoTarget is in sudo's own PATH, so `sudo burrow-g` finds the wrapper there.
const sudoTarget = "/usr/local/bin/burrow-g"

const wrapperMark = "# Written by 'burrow-g setup-sudo'."

// setupSudo installs a wrapper at target that runs binary with the user's
// PATH added, so agents installed in the user's home are found as root too.
func setupSudo(binary, path, target, sudoUser string, run func(name string, args ...string) error) error {
	if sudoUser != "" {
		return errors.New("run setup-sudo without sudo, it asks for your password itself")
	}
	if binary == target {
		return fmt.Errorf("burrow-G is already in %s, so 'sudo burrow-g' already works", target)
	}
	if _, err := os.Lstat(target); err == nil {
		data, err := os.ReadFile(target)
		if err != nil || !strings.Contains(string(data), wrapperMark) {
			return fmt.Errorf("%s already exists and was not made by burrow-G, leaving it alone", target)
		}
	}

	tmp, err := os.CreateTemp("", "burrow-g-wrapper-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(sudoWrapper(binary, path)); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	fmt.Printf("Installing %s (sudo will ask for your password)\n", target)
	return run("sudo", "sh", "-c", `mkdir -p "$(dirname "$1")" && install -m 755 "$2" "$1"`, "sh", target, tmp.Name())
}

// sudoWrapper adds the user's PATH after sudo's own, so system programs still win.
func sudoWrapper(binary, path string) string {
	return "#!/bin/sh\n" +
		wrapperMark + " Run it again if burrow-G moves.\n" +
		`PATH="$PATH":` + shellQuote(path) + "\n" +
		"exec " + shellQuote(binary) + ` "$@"` + "\n"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runSetupSudo is the `burrow-g setup-sudo` command.
func runSetupSudo() error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	if binary, err = filepath.EvalSymlinks(binary); err != nil {
		return err
	}
	if err := setupSudo(binary, os.Getenv("PATH"), sudoTarget, os.Getenv("SUDO_USER"), runAttached); err != nil {
		return err
	}
	fmt.Println("Done. Now 'sudo burrow-g' works.")
	return nil
}

// runAttached runs a command in this terminal, so sudo can ask for the password.
func runAttached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// mergeHomes pairs each of root's agent sources with the caller's, so
// `sudo burrow-g` shows both users' sessions.
func mergeHomes(mine, theirs []session.Source, me, them string) []session.Source {
	merged := make([]session.Source, len(mine))
	for i := range mine {
		merged[i] = session.Merged{{Owner: me, Source: mine[i]}, {Owner: them, Source: theirs[i]}}
	}
	return merged
}

// asOwner resumes another user's session as that user, so the agent finds it
// in their home and never writes root-owned files there. me is "" outside sudo.
func asOwner(s session.Session, argv []string, me, path string) []string {
	if me == "" || s.Owner == "" || s.Owner == me {
		return argv
	}
	return append([]string{"sudo", "-u", s.Owner, "-H", "--", "env", "PATH=" + path}, argv...)
}
