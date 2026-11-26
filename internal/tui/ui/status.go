package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// Kind is what a status line reports.
type Kind int

const (
	OK Kind = iota
	Busy
	Error
)

// Status is the status line's state. Frame is the spinner's current frame
// while Busy; empty draws the plain dot.
type Status struct {
	Kind  Kind
	Text  string
	Frame string
}

// StatusDot draws the dot (or spinner frame) and text, colored by kind.
// The text itself carries the meaning too, so nothing is lost without color.
func StatusDot(t theme.Theme, s Status) string {
	style, glyph := t.OK, t.Glyph.Dot
	switch s.Kind {
	case Busy:
		style = t.Busy
		if s.Frame != "" {
			glyph = s.Frame
		}
	case Error:
		style = t.Error
	}
	return style.Render(glyph + " " + s.Text)
}

// StatusLine draws the status on the left and the key hints flush against the
// right edge, width cells in all, so the hints stay put while the status comes
// and goes. When it doesn't fit, hints are left out from the end (they are
// listed most important first); with none left the status itself is cut.
func StatusLine(t theme.Theme, s Status, width int, keys ...key.Binding) string {
	left := ""
	if s.Text != "" {
		left = " " + StatusDot(t, s)
	}
	for n := len(keys); n > 0; n-- {
		right := KeyHints(t, keys[:n]...)
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 1; gap >= 2 {
			return left + strings.Repeat(" ", gap) + right + " "
		}
	}
	if lipgloss.Width(left) > width {
		return ansi.Truncate(left, width, t.Glyph.Ellipsis)
	}
	return left + strings.Repeat(" ", width-lipgloss.Width(left))
}
