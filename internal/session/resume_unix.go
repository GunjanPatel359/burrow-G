//go:build !windows

package session

import (
	"os"
	"syscall"
)

// run replaces this program with the agent.
func run(path string, argv []string) error {
	return syscall.Exec(path, argv, os.Environ())
}
