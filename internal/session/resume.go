package session

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Resume moves into the session's folder and replaces this program with
// the agent CLI. It only returns if something went wrong.
func Resume(s Session, argv []string) error {
	if err := CheckDir(s); err != nil {
		return err
	}
	path, err := CheckCommand(argv[0], exec.LookPath)
	if err != nil {
		return err
	}
	if err := os.Chdir(s.Dir); err != nil {
		return err
	}
	return syscall.Exec(path, argv, os.Environ())
}

// CheckDir fails if the session's folder is gone.
func CheckDir(s Session) error {
	if _, err := os.Stat(s.Dir); err != nil {
		return fmt.Errorf("folder no longer exists: %s", s.Dir)
	}
	return nil
}

// CheckCommand finds the agent's CLI, saying plainly when it isn't installed.
func CheckCommand(name string, lookPath func(string) (string, error)) (string, error) {
	path, err := lookPath(name)
	if errors.Is(err, exec.ErrNotFound) {
		return "", notInstalledError{name: name, err: err}
	}
	return path, err
}

// notInstalledError keeps the lookup error in the chain: the sudo hint
// (cmd/burrow-g/sudo.go) looks for exec.ErrNotFound.
type notInstalledError struct {
	name string
	err  error
}

func (e notInstalledError) Error() string {
	return e.name + " is not installed (not found in PATH)"
}

func (e notInstalledError) Unwrap() error { return e.err }
