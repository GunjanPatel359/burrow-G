package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestPrintList(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	s := session.Session{
		Agent:    "claude",
		ID:       "2b3f3fde-f377-462d-8448-c0ddeed18392",
		Title:    "New title",
		Dir:      "/home/u/proj",
		LastUsed: now.Add(-2 * time.Hour),
	}
	marked := session.Session{Agent: "claude", ID: "bbb", Title: "Starred", LastUsed: now, Bookmarked: true}

	var out bytes.Buffer
	printList(&out, []session.Session{marked, s}, "/home/u", now)
	lines := strings.Split(out.String(), "\n")

	if !strings.HasPrefix(lines[0], "★ Starred") {
		t.Errorf("bookmarked row should start with ★: %q", lines[0])
	}
	order := regexp.MustCompile(`New title.*claude.*~/proj.*2 hours ago.*2b3f3fde`)
	if !order.MatchString(lines[1]) {
		t.Errorf("row has wrong content or order: %q", lines[1])
	}
}

type fakeSource struct {
	name string
	list []session.Session
	err  error
}

func (f fakeSource) Name() string                           { return f.name }
func (f fakeSource) List() ([]session.Session, error)       { return f.list, f.err }
func (f fakeSource) ResumeCommand(session.Session) []string { return nil }
func (f fakeSource) Delete(session.Session, string) error   { return nil }
func (fakeSource) Export(session.Session, *portable.Writer) error {
	return session.ErrUnsupported
}

func TestRefreshAll_SkipsBrokenSource(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	broken := fakeSource{name: "fake", err: errors.New("db missing")}
	good := fakeSource{name: "good", list: []session.Session{{Agent: "good", ID: "a"}}}

	var warn bytes.Buffer
	got := refreshAll(store, []session.Source{broken, good}, &warn)

	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("got %+v, want the one session from 'good'", got)
	}
	if !strings.Contains(warn.String(), "warning: fake: db missing") {
		t.Errorf("warning = %q", warn.String())
	}
}

func TestPickSession(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Save("claude", []session.Session{
		{Agent: "claude", ID: "2b3f3fde-1111", Title: "One"},
		{Agent: "claude", ID: "2b3f9999-2222", Title: "Two"},
	}); err != nil {
		t.Fatal(err)
	}
	sources := []session.Source{fakeSource{name: "claude"}}

	s, src, err := pickSession(store, sources, "2b3f3")
	if err != nil || s.Title != "One" || src.Name() != "claude" {
		t.Errorf("unique prefix: got %+v, %v, %v", s, src, err)
	}
	if _, _, err := pickSession(store, sources, "2b3f"); err == nil || !strings.Contains(err.Error(), "2 sessions match") {
		t.Errorf("ambiguous prefix: err = %v", err)
	}
	if _, _, err := pickSession(store, sources, "zzz"); err == nil || !strings.Contains(err.Error(), "no session") {
		t.Errorf("no match: err = %v", err)
	}
}

func TestUsage(t *testing.T) {
	var out bytes.Buffer
	printUsage(&out)
	for _, want := range []string{"burrow-G", "burrow-g list", "burrow-g resume <id>", "burrow-g version", "burrow-g setup-sudo", "enter", "~/.local/share/burrow-g"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage is missing %q:\n%s", want, out.String())
		}
	}
}

func TestVersionString(t *testing.T) {
	if got := versionString(); got != "burrow-G "+version {
		t.Errorf("versionString() = %q, want %q", got, "burrow-G "+version)
	}
}

func TestUsageKeyColumnsAligned(t *testing.T) {
	_, ui, _ := strings.Cut(usage, "In the UI:\n")
	ui, _, _ = strings.Cut(ui, "\n\n")
	field := regexp.MustCompile(`\S+( \S+)*`)
	col := -1
	for row := range strings.SplitSeq(ui, "\n") {
		idx := field.FindAllStringIndex(row, -1)
		if len(idx) < 3 {
			t.Fatalf("row %q has fewer than 3 columns", row)
		}
		c := utf8.RuneCountInString(row[:idx[2][0]])
		if col == -1 {
			col = c
		}
		if c != col {
			t.Errorf("row %q: second key at column %d, want %d", row, c, col)
		}
	}
}
