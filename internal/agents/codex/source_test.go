package codex

import (
	"errors"
	"slices"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

func TestList(t *testing.T) {
	src := Source{Home: "testdata"}
	if src.Name() != "codex" {
		t.Errorf("Name() = %q, want codex", src.Name())
	}

	got, err := src.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sessions, want 2 (subagent skipped): %+v", len(got), got)
	}

	byID := map[string]session.Session{}
	for _, s := range got {
		byID[s.ID] = s
	}
	if s := byID["c1"]; s.Agent != "codex" || s.Title != "add retry logic" || s.Dir != "/tmp/c" {
		t.Errorf("c1 = %+v", s)
	}
	if s := byID["c2"]; s.Title != "Named thread" || s.Dir != "/tmp/d" {
		t.Errorf("c2 = %+v", s)
	}
}

func TestList_NoSessionsFolder(t *testing.T) {
	if _, err := (Source{Home: t.TempDir()}).List(); !errors.Is(err, session.ErrMissing) {
		t.Errorf("err = %v, want session.ErrMissing", err)
	}
}

func TestResumeCommand(t *testing.T) {
	got := Source{}.ResumeCommand(session.Session{ID: "c1"})
	want := []string{"codex", "resume", "c1"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDelete_NotSupported(t *testing.T) {
	if err := (Source{}).Delete(session.Session{ID: "c1"}, t.TempDir()); err == nil {
		t.Error("want an error: delete is not supported for codex yet")
	}
}
