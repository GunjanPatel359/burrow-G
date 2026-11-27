package session

import (
	"strings"
	"testing"
)

func TestResume_MissingFolder(t *testing.T) {
	err := Resume(Session{Dir: "/nonexistent/dir"}, []string{"true"})
	if err == nil || !strings.Contains(err.Error(), "folder no longer exists") {
		t.Errorf("err = %v, want 'folder no longer exists'", err)
	}
}

func TestCheckDir(t *testing.T) {
	if err := CheckDir(Session{Dir: t.TempDir()}); err != nil {
		t.Errorf("existing folder: %v", err)
	}
	if err := CheckDir(Session{Dir: "/nonexistent/dir"}); err == nil || !strings.Contains(err.Error(), "folder no longer exists") {
		t.Errorf("missing folder: err = %v", err)
	}
}

func TestResume_UnknownCommand(t *testing.T) {
	err := Resume(Session{Dir: t.TempDir()}, []string{"no-such-agent-cli-xyz"})
	if err == nil {
		t.Error("want an error for a command that isn't installed")
	}
}
