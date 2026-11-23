package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// background sets the terminal's own background colour, so a theme fills the
// whole window, and puts the original back when burrow-G is done. Terminals
// that don't support it ignore both.
type background struct {
	out  *termenv.Output
	orig termenv.Color
}

func newBackground() background {
	out := lipgloss.DefaultRenderer().Output()
	return background{out: out, orig: out.BackgroundColor()}
}

func (b background) set(c lipgloss.Color) {
	if c != "" {
		b.out.SetBackgroundColor(b.out.Color(string(c)))
	}
}

func (b background) restore() {
	if rgb, ok := b.orig.(termenv.RGBColor); ok {
		b.out.SetBackgroundColor(rgb)
		return
	}
	b.out.WriteString("\x1b]111\x07") // reset to the terminal's default
}
