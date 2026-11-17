package session

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// run starts the agent and exits with its exit code once it is done.
// Windows can't replace a running program like syscall.Exec does on Unix.
func run(path string, argv []string) error {
	signal.Ignore(os.Interrupt) // ctrl+c is for the agent, not for us

	cmd := exec.Command(path, argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()

	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
