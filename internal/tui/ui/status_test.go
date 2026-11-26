package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestStatusDot(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	cases := []struct {
		s    Status
		want string
	}{
		{Status{Kind: OK, Text: "refreshed"}, "● refreshed"},
		{Status{Kind: Busy, Text: "refreshing…"}, "● refreshing…"},
		{Status{Kind: Busy, Text: "refreshing…", Frame: "⠙"}, "⠙ refreshing…"},
		{Status{Kind: Error, Text: "error: boom"}, "● error: boom"},
	}
	for _, c := range cases {
		if got := visible(StatusDot(th, c.s)); got != c.want {
			t.Errorf("StatusDot(%+v) = %q, want %q", c.s, got, c.want)
		}
	}
	if StatusDot(th, cases[0].s) == StatusDot(th, Status{Kind: Error, Text: "refreshed"}) {
		t.Error("OK and Error should differ in color")
	}
	if got := visible(StatusDot(theme.Get(theme.Default, theme.Env{ASCII: true}), cases[0].s)); got != "o refreshed" {
		t.Errorf("ASCII dot = %q", got)
	}
}

func statusKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func TestStatusLineHintsFlushRight(t *testing.T) {
	th := plainTheme()
	for _, s := range []Status{{}, {Kind: OK, Text: "exported \"x\""}} {
		got := visible(StatusLine(th, s, 60, statusKeys()...))
		if ansi.StringWidth(got) != 60 || !strings.HasSuffix(got, "q quit ") {
			t.Errorf("status %q: line %q is %d cells, want 60 ending in the hints", s.Text, got, ansi.StringWidth(got))
		}
	}
}

func TestStatusLineDropsHintsFromTheEnd(t *testing.T) {
	got := visible(StatusLine(plainTheme(), Status{Kind: OK, Text: "refreshed"}, 30, statusKeys()...))
	if ansi.StringWidth(got) > 30 || !strings.Contains(got, "refreshed") || !strings.Contains(got, "r refresh") || strings.Contains(got, "quit") {
		t.Errorf("line = %q (%d cells), want the status and the first hint in 30", got, ansi.StringWidth(got))
	}
}

func TestStatusLineCutsALongStatus(t *testing.T) {
	long := Status{Kind: Error, Text: "error: " + strings.Repeat("x", 80)}
	if got := visible(StatusLine(plainTheme(), long, 40, statusKeys()...)); ansi.StringWidth(got) > 40 {
		t.Errorf("line = %q is %d cells, want at most 40", got, ansi.StringWidth(got))
	}
}
