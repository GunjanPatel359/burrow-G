package session

import (
	"fmt"
	"os"
	"os/exec"
)

// Resume moves into the session's folder and hands the terminal to the
// agent CLI. It only returns if something went wrong.
func Resume(s Session, argv []string) error {
	if err := CheckDir(s); err != nil {
		return err
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	if err := os.Chdir(s.Dir); err != nil {
		return err
	}
	return run(path, argv)
}

// CheckDir fails if the session's folder is gone.
func CheckDir(s Session) error {
	if _, err := os.Stat(s.Dir); err != nil {
		return fmt.Errorf("folder no longer exists: %s", s.Dir)
	}
	return nil
}
