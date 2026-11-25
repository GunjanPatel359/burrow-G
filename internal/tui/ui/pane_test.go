package ui

import (
	"testing"

	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestPaneHeader(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	focused, other := PaneHeader(th, "Agents", true), PaneHeader(th, "Agents", false)
	if visible(focused) != " Agents" || visible(other) != " Agents" {
		t.Errorf("headers = %q, %q, want \" Agents\"", visible(focused), visible(other))
	}
	if focused == other {
		t.Error("focused and unfocused headers look the same")
	}
}

func TestRules(t *testing.T) {
	th := plainTheme()
	if got := visible(Divider(th, 5)); got != "─────" {
		t.Errorf("Divider = %q", got)
	}
	if got := visible(VRule(th, 3)); got != "│\n│\n│" {
		t.Errorf("VRule = %q", got)
	}
	ascii := theme.Get(theme.Default, theme.Env{ASCII: true})
	if got := visible(Divider(ascii, 3)) + visible(VRule(ascii, 1)); got != "---|" {
		t.Errorf("ASCII rules = %q", got)
	}
	if visible(Divider(th, -1)) != "" || VRule(th, 0) != "" {
		t.Error("rules should be empty at zero size")
	}
}
