package ui

import "github.com/GunjanPatel359/burrow-G/internal/tui/theme"

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
