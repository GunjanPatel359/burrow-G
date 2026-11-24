package sessionlist

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var now = time.Now()

func newTest(width int, list ...session.Session) Model {
	m := New(theme.Get(theme.Default), "/home/u")
	m.SetSize(width, 10)
	m.SetSessions(list)
	return m
}

// send updates m and feeds back whatever its command produces, like Bubble Tea does.
func send(m Model, msg tea.Msg) Model {
	m, cmd := m.Update(msg)
	if cmd == nil {
		return m
	}
	out := cmd()
	if batch, ok := out.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				m, _ = m.Update(c())
			}
		}
		return m
	}
	m, _ = m.Update(out)
	return m
}

func TestRow(t *testing.T) {
	s := session.Session{Title: "New title", LastUsed: now.Add(-2 * time.Hour), Bookmarked: true}
	first := strings.Split(newTest(60, s).View(), "\n")[1]

	for _, want := range []string{"★", "New title", "2 hours ago"} {
		if !strings.Contains(first, want) {
			t.Errorf("row %q is missing %q", first, want)
		}
	}
}

func TestLongTitleFits(t *testing.T) {
	s := session.Session{Title: strings.Repeat("very long title ", 10), LastUsed: now}
	first, _, _ := strings.Cut(newTest(40, s).View(), "\n")
	if w := len([]rune(first)); w > 40 {
		t.Errorf("row is %d characters, want at most 40: %q", w, first)
	}
}

func TestSelected(t *testing.T) {
	m := newTest(60)
	if _, ok := m.Selected(); ok {
		t.Error("empty list should have no selection")
	}

	m.SetSessions([]session.Session{{ID: "aaa", LastUsed: now}, {ID: "bbb", LastUsed: now}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s, ok := m.Selected(); !ok || s.ID != "bbb" {
		t.Errorf("Selected() = %+v, %v; want bbb", s, ok)
	}
	if m.Len() != 2 {
		t.Errorf("Len() = %d, want 2", m.Len())
	}
}

func TestSearchBoxOnlyWhileSearching(t *testing.T) {
	m := newTest(60, session.Session{Title: "Fix auth", Dir: "/home/u/proj", LastUsed: now})

	if !strings.Contains(strings.Split(m.View(), "\n")[1], "Fix auth") {
		t.Fatalf("second line should be the first session, got %q", m.View())
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !m.Searching() || !strings.Contains(m.View(), "Filter:") {
		t.Error("search box should appear after /")
	}
}

func TestSearchMatchesFolder(t *testing.T) {
	m := newTest(60,
		session.Session{ID: "a", Title: "One", Dir: "/home/u/alpha", LastUsed: now},
		session.Session{ID: "b", Title: "Two", Dir: "/home/u/beta", LastUsed: now},
	)
	for _, r := range "/beta" {
		m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if s, ok := m.Selected(); !ok || s.ID != "b" {
		t.Errorf("searching a folder name should find it, got %+v", s)
	}
}

func TestSelect(t *testing.T) {
	a := session.Session{Agent: "claude", ID: "a", LastUsed: now}
	b := session.Session{Agent: "claude", ID: "b", LastUsed: now}
	c := session.Session{Agent: "claude", ID: "c", LastUsed: now}
	m := newTest(60, a, b, c)

	m.Select(c.Key())
	if s, _ := m.Selected(); s.ID != "c" {
		t.Errorf("Select(c) left the cursor on %q", s.ID)
	}
	m.Select("claude:gone")
	if s, _ := m.Selected(); s.ID != "c" {
		t.Errorf("Select(unknown) moved the cursor to %q", s.ID)
	}
}

func TestSetSessions_KeepsSearchResults(t *testing.T) {
	one := session.Session{ID: "a", Title: "One", Dir: "/home/u/alpha", LastUsed: now}
	two := session.Session{ID: "b", Title: "Two", Dir: "/home/u/beta", LastUsed: now}
	m := newTest(60, one, two)
	for _, r := range "/beta" {
		m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	m.SetSessions([]session.Session{one, two})
	if s, ok := m.Selected(); !ok || s.ID != "b" {
		t.Errorf("after reloading, the search should still show beta; selected %+v, %v", s, ok)
	}
}

func TestSearchLineAlwaysShown(t *testing.T) {
	m := newTest(60,
		session.Session{ID: "a", Title: "One", LastUsed: now},
		session.Session{ID: "b", Title: "Two", LastUsed: now},
	)
	if first, _, _ := strings.Cut(m.View(), "\n"); !strings.Contains(first, "/ search") {
		t.Fatalf("first line should be the search hint, got %q", first)
	}
	before := strings.Count(m.View(), "\n")
	for _, r := range "/x" {
		m = send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if strings.Contains(m.View(), "/ search") {
		t.Error("hint should give way to the filter line")
	}
	if got := strings.Count(m.View(), "\n"); got != before {
		t.Errorf("height changed: %d -> %d lines", before, got)
	}
}
