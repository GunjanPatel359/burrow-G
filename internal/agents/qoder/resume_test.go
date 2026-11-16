package qoder

import (
	"slices"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestResumeCommand(t *testing.T) {
	got := Source{}.ResumeCommand(session.Session{ID: "abc"})
	want := []string{"qodercli", "--resume", "abc"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
