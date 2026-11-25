package ui

import (
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
