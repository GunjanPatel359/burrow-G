package ui

import (
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// PaneHeader is a pane's title line: Accent bold when the pane has focus,
// Muted bold when it does not.
func PaneHeader(t theme.Theme, title string, focused bool) string {
	style := t.Muted.Bold(true)
	if focused {
		style = t.Header
	}
	return style.Render(gutter() + title)
}

// Divider is a horizontal rule across width cells.
func Divider(t theme.Theme, width int) string {
	return t.Border.Render(strings.Repeat(t.Glyph.HLine, max(width, 0)))
}

// VRule is a vertical rule height lines tall.
func VRule(t theme.Theme, height int) string {
	if height <= 0 {
		return ""
	}
	return t.Border.Render(strings.TrimSuffix(strings.Repeat(t.Glyph.VLine+"\n", height), "\n"))
}

func gutter() string { return strings.Repeat(" ", theme.Gutter) }
