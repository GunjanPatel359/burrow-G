package transcript

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

type fakeAgent struct {
	name string
	bin  string
	tr   session.Transcript
}

func (f fakeAgent) Name() string                                           { return f.name }
func (f fakeAgent) List() ([]session.Session, error)                       { return nil, nil }
func (f fakeAgent) ResumeCommand(session.Session) []string                 { return nil }
func (f fakeAgent) Delete(session.Session, string) error                   { return nil }
func (f fakeAgent) Transcript(session.Session) (session.Transcript, error) { return f.tr, nil }

func (f fakeAgent) StartCommand(dir, prompt string) []string {
	if f.bin == "" {
		return nil
	}
	return []string{f.bin, prompt}
}

func found(string) (string, error)    { return "/usr/bin/x", nil }
func notFound(string) (string, error) { return "", errors.New("not found") }

func TestHandoff(t *testing.T) {
	tr := sample()
	tr.Session.Dir = t.TempDir()
	src := fakeAgent{name: "claude", bin: "claude", tr: tr}
	dst := fakeAgent{name: "codex", bin: "codex"}

	argv, st, err := Handoff(src, dst, tr.Session, found)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := Prompt(tr)
	if len(argv) != 2 || argv[0] != "codex" || argv[1] != want {
		t.Errorf("argv = %q", argv)
	}
	if st.Messages != 3 || st.Bytes != len(want) {
		t.Errorf("stats = %+v", st)
	}
}

func TestHandoff_Refusals(t *testing.T) {
	tr := sample()
	tr.Session.Dir = t.TempDir()
	gone := tr.Session
	gone.Dir = filepath.Join(t.TempDir(), "gone")
	empty := fakeAgent{name: "claude", bin: "claude", tr: session.Transcript{Session: tr.Session}}

	tests := []struct {
		name     string
		src, dst fakeAgent
		s        session.Session
		lookPath func(string) (string, error)
		want     string
	}{
		{"folder gone", fakeAgent{tr: tr}, fakeAgent{name: "codex", bin: "codex"}, gone, found, "folder no longer exists: " + gone.Dir},
		{"not installed", fakeAgent{tr: tr}, fakeAgent{name: "codex", bin: "codex"}, tr.Session, notFound, "codex is not installed (not found on PATH)"},
		{"no first prompt", fakeAgent{tr: tr}, fakeAgent{name: "qoder"}, tr.Session, found, "qoder can't start with a first prompt yet"},
		{"no messages", empty, fakeAgent{name: "codex", bin: "codex"}, tr.Session, found, "no messages found in claude session abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv, _, err := Handoff(tt.src, tt.dst, tt.s, tt.lookPath)
			if err == nil || err.Error() != tt.want {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
			if len(argv) != 0 {
				t.Errorf("argv = %q, want none", argv)
			}
		})
	}
}

func TestTargets(t *testing.T) {
	sources := []session.Source{
		fakeAgent{name: "claude", bin: "claude"},
		fakeAgent{name: "qoder"},
		fakeAgent{name: "codex", bin: "codex"},
	}
	onlyClaude := func(bin string) (string, error) {
		if bin == "claude" {
			return "/bin/claude", nil
		}
		return "", errors.New("no")
	}
	got := Targets(sources, onlyClaude)
	if len(got) != 1 || got[0].Name() != "claude" {
		t.Errorf("targets = %v", got)
	}
}

func TestReport(t *testing.T) {
	got := Report("codex", Stats{Messages: 12, Bytes: 38_200, Omitted: 4, Masked: 1, Skipped: 2})
	want := "handed off to codex: 12 messages, 38.2 KB prompt, 4 omitted, 1 secrets masked, 2 entries skipped"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if strings.Contains(ExportReport(Stats{Messages: 3}), "skipped") {
		t.Error("no skipped part when nothing was skipped")
	}
}
