package session

import (
	"errors"
	"os/exec"
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
	if err == nil || err.Error() != "no-such-agent-cli-xyz is not installed (not found in PATH)" {
		t.Errorf("err = %v, want 'no-such-agent-cli-xyz is not installed (not found in PATH)'", err)
	}
}

func TestCheckCommand(t *testing.T) {
	denied := errors.New("permission denied")
	tests := []struct {
		name    string
		lookErr error
		want    string // error text, "" for none
	}{
		{"installed", nil, ""},
		{"not installed", exec.ErrNotFound, "codebuff is not installed (not found in PATH)"},
		{"other error", denied, "permission denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := CheckCommand("codebuff", func(name string) (string, error) {
				if tt.lookErr != nil {
					return "", &exec.Error{Name: name, Err: tt.lookErr}
				}
				return "/bin/" + name, nil
			})
			if tt.want == "" {
				if err != nil || path != "/bin/codebuff" {
					t.Errorf("got %q, %v; want /bin/codebuff", path, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
			// The sudo hint looks for exec.ErrNotFound in the chain.
			if !errors.Is(err, tt.lookErr) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.lookErr, err)
			}
		})
	}
}
