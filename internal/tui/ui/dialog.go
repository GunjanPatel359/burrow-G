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
	return frame(t, title, keys, bodyWidth, func(b *box) {
		for _, line := range body {
			b.text(t.Text, line)
		}
	})
}

// Choice is a Dialog that offers choices: lead lines, then one row per
// choice, the selected one marked and highlighted like a list row.
func Choice(t theme.Theme, title string, lead, choices []string, selected int, keys []key.Binding, bodyWidth int) string {
	return frame(t, title, keys, bodyWidth, func(b *box) {
		for _, line := range lead {
			b.text(t.Text, line)
		}
		if len(lead) > 0 {
			b.row("")
		}
		for i, c := range choices {
			b.row(SurfaceRow(t, []theme.Part{{Text: c}}, nil, b.inner, i == selected))
		}
	})
}

// box collects a dialog's rows between its side rules.
type box struct {
	t     theme.Theme
	inner int // cells between the padding
	rows  []string
}

func (b *box) row(styled string) {
	fill, side := b.t.Surface, b.t.Border.Inherit(b.t.Surface).Render(b.t.Glyph.VLine)
	pad := strings.Repeat(" ", max(b.inner-ansi.StringWidth(styled), 0)+1)
	b.rows = append(b.rows, side+fill.Render(" ")+styled+fill.Render(pad)+side)
}

// text wraps plain text first, so a color never spans a line break.
func (b *box) text(style lipgloss.Style, s string) {
	for l := range strings.SplitSeq(ansi.Wrap(s, b.inner, ""), "\n") {
		b.row(style.Inherit(b.t.Surface).Render(l))
	}
}

// frame draws the rounded box: title, the rows body adds, key hints.
func frame(t theme.Theme, title string, keys []key.Binding, bodyWidth int, body func(*box)) string {
	width := max(24, min(50, bodyWidth-4))
	b := &box{t: t, inner: width - 4} // two frame cells, one padding cell each side
	b.text(t.Header, title)
	b.row("")
	body(b)
	b.row("")
	for _, line := range hintLines(t.Glyph, keys, b.inner) {
		b.row(keyHints(t, t.Surface, line))
	}

	c := t.Glyph.Corners
	edge := strings.Repeat(t.Glyph.HLine, width-2)
	border := t.Border.Inherit(t.Surface)
	return border.Render(c[0]+edge+c[1]) + "\n" + strings.Join(b.rows, "\n") + "\n" + border.Render(c[2]+edge+c[3])
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
