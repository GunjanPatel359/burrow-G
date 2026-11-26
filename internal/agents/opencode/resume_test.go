package opencode

import (
	"slices"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestResumeCommand(t *testing.T) {
	got := Source{}.ResumeCommand(session.Session{ID: "abc"})
	want := []string{"opencode", "--session", "abc"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The command each CLI's --help documents for a first prompt (spec 0004).
func TestStartCommand(t *testing.T) {
	if got, want := (Source{}).StartCommand("hi"), []string{"opencode", "--prompt", "hi"}; !slices.Equal(got, want) {
		t.Errorf("StartCommand() = %q, want %q", got, want)
	}
}
