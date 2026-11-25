package theme

import (
	"strings"

	"github.com/muesli/termenv"
)

// Depth is how many colors the terminal can show.
type Depth int

const (
	TrueColor Depth = iota
	Color256
	Color16
	NoColor // NO_COLOR is set, or the terminal has no colors
)

// Env is what the terminal can draw. The zero value is truecolor with Unicode.
type Env struct {
	Depth Depth
	ASCII bool // the locale is not UTF-8, so glyphs fall back to plain ASCII
}

// DetectEnv reads the color profile the terminal reports and the locale.
func DetectEnv(p termenv.Profile, getenv func(string) string) Env {
	return Env{Depth: DepthOf(p), ASCII: asciiLocale(getenv)}
}

// DepthOf maps a termenv profile to a Depth.
func DepthOf(p termenv.Profile) Depth {
	switch p {
	case termenv.TrueColor:
		return TrueColor
	case termenv.ANSI256:
		return Color256
	case termenv.ANSI:
		return Color16
	}
	return NoColor
}

// Profile is the profile the Lip Gloss renderer must use for this depth.
// NoColor still uses ANSI, because at Ascii termenv also drops bold, faint,
// reverse and underline, which are all NoColor has left. Its styles carry no
// colors, so nothing colored is drawn.
func (d Depth) Profile() termenv.Profile {
	switch d {
	case TrueColor:
		return termenv.TrueColor
	case Color256:
		return termenv.ANSI256
	}
	return termenv.ANSI
}

// Paints reports whether themes set the window background at this depth. At
// 16 colors the terminal's own background is kept, so its colors still match.
func (d Depth) Paints() bool { return d == TrueColor || d == Color256 }

// asciiLocale is true when the first set of LC_ALL, LC_CTYPE and LANG is not
// UTF-8. None set means UTF-8.
func asciiLocale(getenv func(string) string) bool {
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := strings.ToLower(getenv(k)); v != "" {
			return !strings.Contains(v, "utf-8") && !strings.Contains(v, "utf8")
		}
	}
	return false
}
