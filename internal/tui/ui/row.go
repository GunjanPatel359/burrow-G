// Package ui draws the shared pieces of the screen from a theme.Theme, so
// components never pick colors or glyphs themselves.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// Row draws one list row of the given width: left text, right text at the end,
// and a marker plus highlight when selected. Each part is drawn on top of the
// row's style, so the highlight also runs through colored parts like the star.
func Row(t theme.Theme, left, right []theme.Part, width int, selected, active bool) string {
	base, marker := t.Text, " "
	if selected {
		marker = t.Marker.Render(t.Glyph.Marker)
	}
	if selected && active {
		base = t.Selected
	}
	return marker + line(base, left, right, width) + " "
}

// SurfaceRow is a Row on the Surface background, for the command bar
// suggestions. The fill runs through the gap and the padding too.
func SurfaceRow(t theme.Theme, left, right []theme.Part, width int, selected bool) string {
	base := t.Surface.Inherit(t.Text)
	marker := base.Render(" ")
	if selected {
		base = t.Selected
		marker = t.Marker.Inherit(t.Surface).Render(t.Glyph.Marker)
	}
	return marker + line(base, left, right, width) + base.Render(" ")
}

// line is a row between its marker column and its last cell: parts, a gap, parts.
func line(base lipgloss.Style, left, right []theme.Part, width int) string {
	draw := func(parts []theme.Part) (string, int) {
		var b strings.Builder
		w := 0
		for _, p := range parts {
			b.WriteString(p.Style.Inherit(base).Render(p.Text))
			w += lipgloss.Width(p.Text)
		}
		return b.String(), w
	}
	l, lw := draw(left)
	r, rw := draw(right)
	gap := base.Render(strings.Repeat(" ", max(width-2-lw-rw, 1)))
	return l + gap + r
}
