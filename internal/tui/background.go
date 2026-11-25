package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// background sets the terminal's own background colour, so a theme fills the
// whole window, and puts the original back when burrow-G is done. Terminals
// that don't support it ignore both.
type background struct {
	out     *termenv.Output
	orig    termenv.Color
	painted bool
}

// newBackground asks the terminal for its background. Call it before the
// program starts: the answer is read from the terminal and would race with
// Bubble Tea's input.
func newBackground() *background {
	out := lipgloss.DefaultRenderer().Output()
	return &background{out: out, orig: out.BackgroundColor()}
}

func (b *background) set(c lipgloss.Color) {
	if c != "" {
		b.out.SetBackgroundColor(b.out.Color(string(c)))
		b.painted = true
	}
}

// restore runs only after a paint, so depths that keep the terminal's own
// background never touch it.
func (b *background) restore() {
	if !b.painted {
		return
	}
	if rgb, ok := b.orig.(termenv.RGBColor); ok {
		b.out.SetBackgroundColor(rgb)
		return
	}
	b.out.WriteString("\x1b]111\x07") // reset to the terminal's default
}

// dark reports whether the terminal's background is dark. No answer (tmux,
// screen, or a terminal that ignores the query) counts as dark.
func (b *background) dark() bool {
	_, _, l := termenv.ConvertToRGB(b.orig).Hsl()
	return l < 0.5
}
