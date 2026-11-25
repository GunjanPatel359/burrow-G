package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// EmptyState fills a width × height area with a message and a hint below it,
// centered. Lines that don't fit are cut; with one line only the message shows.
func EmptyState(t theme.Theme, message, hint string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	fit := func(style lipgloss.Style, s string) string {
		return style.Render(ansi.Truncate(s, width, t.Glyph.Ellipsis))
	}
	lines := []string{fit(t.Text, message)}
	if hint != "" && height > 1 {
		lines = append(lines, fit(t.Muted, hint))
	}
	box := lipgloss.JoinVertical(lipgloss.Center, lines...)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
