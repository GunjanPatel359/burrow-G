package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// withProfile sets the renderer profile for one test, as tui.Run does.
func withProfile(t *testing.T, p termenv.Profile) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(p)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func plainTheme() theme.Theme { return theme.Get(theme.Default, theme.Env{}) }

func visible(s string) string { return ansi.Strip(s) }
