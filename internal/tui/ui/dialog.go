package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// Dialog is a framed box for a question: title, body, then key hints. bodyWidth
// is the width of the area it is shown over. The box is 24 to 50 cells wide and
// never cuts its body; long lines wrap.
func Dialog(t theme.Theme, title string, body []string, keys []key.Binding, bodyWidth int) string {
	width := max(24, min(50, bodyWidth-4))
	inner := width - 4 // two frame cells, one padding cell each side
	fill := t.Surface
	frame := t.Border.Inherit(fill)
	side := frame.Render(t.Glyph.VLine)

	var rows []string
	row := func(styled string) {
		pad := strings.Repeat(" ", max(inner-ansi.StringWidth(styled), 0)+1)
		rows = append(rows, side+fill.Render(" ")+styled+fill.Render(pad)+side)
	}
	// Wrap the plain text first, so a color never spans a line break.
	text := func(style lipgloss.Style, s string) {
		for l := range strings.SplitSeq(ansi.Wrap(s, inner, ""), "\n") {
			row(style.Render(l))
		}
	}

	text(t.Header.Inherit(fill), title)
	row("")
	for _, b := range body {
		text(t.Text.Inherit(fill), b)
	}
	row("")
	for _, line := range hintLines(t.Glyph, keys, inner) {
		row(keyHints(t, fill, line))
	}

	c := t.Glyph.Corners
	edge := strings.Repeat(t.Glyph.HLine, width-2)
	return frame.Render(c[0]+edge+c[1]) + "\n" + strings.Join(rows, "\n") + "\n" + frame.Render(c[2]+edge+c[3])
}

// hintLines splits bindings into lines no wider than width, whole hints only.
func hintLines(g theme.GlyphSet, keys []key.Binding, width int) [][]key.Binding {
	var lines [][]key.Binding
	used := 0
	for _, b := range keys {
		w := ansi.StringWidth(HintText(g, b))
		if len(lines) == 0 || used+3+w > width {
			lines = append(lines, nil)
			used = -3 // no separator before the first hint
		}
		lines[len(lines)-1] = append(lines[len(lines)-1], b)
		used += 3 + w
	}
	return lines
}
