// Package launch opens a command in a new terminal tab.
// Each supported terminal lives in its own file (gnome.go, …).
package launch

import (
	"os"
	"os/exec"
)

// Terminal can open a command in a new tab.
type Terminal interface {
	Name() string
	OpenTab(dir, title string, argv []string) error
}

// Detect returns the terminal we are running in, or nil if it is not supported.
func Detect(getenv func(string) string) Terminal {
	if getenv("TERMINAL_EMULATOR") != "" { // IDE terminals (GoLand: JetBrains-JediTerm)
		return nil
	}
	if getenv("GNOME_TERMINAL_SCREEN") != "" || getenv("GNOME_TERMINAL_SERVICE") != "" {
		return gnome{run: start}
	}
	return nil
}

// start launches a program without waiting for it to finish.
func start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = nil, nil
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
