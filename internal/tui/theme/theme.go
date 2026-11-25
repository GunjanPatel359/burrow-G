// Package theme turns a colour palette into ready-to-use styles.
package theme

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name  string
	Env   Env
	Glyph GlyphSet

	Header           lipgloss.Style
	Muted            lipgloss.Style
	Border           lipgloss.Style
	Text             lipgloss.Style
	Star             lipgloss.Style
	Folder           lipgloss.Style
	Label            lipgloss.Style
	Match            lipgloss.Style
	Selected         lipgloss.Style
	SelectedInactive lipgloss.Style
	Surface          lipgloss.Style
	Marker           lipgloss.Style
	OK               lipgloss.Style
	Busy             lipgloss.Style
	Error            lipgloss.Style

	// Background is the window color to paint, or "" when this depth keeps
	// the terminal's own background.
	Background lipgloss.Color
}

// Get returns the named theme drawn for env, or the default one if the name
// is unknown. Each color is resolved here for env.Depth, so the renderer's own
// guess never decides it.
func Get(name string, env Env) Theme {
	p, ok := palettes[name]
	if !ok {
		name, p = Default, palettes[Default]
	}
	fg := func(slot string, c Tone) lipgloss.Style { return env.style(slot, c, false) }
	bg := func(slot string, c Tone) lipgloss.Style { return env.style(slot, c, true) }

	t := Theme{
		Name:  name,
		Env:   env,
		Glyph: Unicode,

		Header:           fg("Accent", p.Accent).Bold(true),
		Muted:            fg("Muted", p.Muted),
		Border:           fg("Border", p.Border),
		Text:             fg("Text", p.Text),
		Star:             fg("Warning", p.Warning),
		Folder:           fg("Info", p.Info),
		Label:            fg("Label", p.Label).Width(LabelWidth),
		Match:            fg("Match", p.Match),
		Selected:         bg("Selected", p.Selected).Inherit(fg("Text", p.Text)).Bold(true),
		SelectedInactive: bg("SelectedInactive", p.SelectedInactive),
		Surface:          bg("Surface", p.Surface),
		Marker:           fg("Accent", p.Accent),
		OK:               fg("Success", p.Success),
		Busy:             fg("Warning", p.Warning),
		Error:            fg("Error", p.Error),
	}
	if env.ASCII {
		t.Glyph = ASCII
	}
	if env.Depth.Paints() {
		t.Background = env.color(p.Background)
	}
	return t
}

// style draws one slot: the palette color at truecolor and 256, the shared
// look at 16 colors and without color.
func (e Env) style(slot string, c Tone, background bool) lipgloss.Style {
	s := lipgloss.NewStyle()
	if e.Depth.Paints() {
		if background {
			return s.Background(e.color(c))
		}
		return s.Foreground(e.color(c))
	}
	l := ansi16[slot]
	if e.Depth == NoColor {
		l = noColor[slot]
	}
	if l.Color != "" {
		s = s.Foreground(lipgloss.Color(l.Color))
	}
	return s.Faint(l.Faint).Bold(l.Bold).Underline(l.Underline).Reverse(l.Reverse)
}

func (e Env) color(c Tone) lipgloss.Color {
	if e.Depth == Color256 {
		return lipgloss.Color(strconv.Itoa(int(c.X256)))
	}
	return lipgloss.Color(c.Hex)
}

// Initial is the theme to start with: a saved, known name wins; otherwise the
// terminal's background picks the dark or light default. The pick is not
// saved, so a later change of terminal is followed.
func Initial(saved string, dark func() bool) string {
	if _, ok := palettes[saved]; ok {
		return saved
	}
	if dark() {
		return Default
	}
	return DefaultLight
}

// Names lists every theme, sorted.
func Names() []string {
	var names []string
	for n := range palettes {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// Part is a piece of row text; an empty Style uses the row's normal colour.
type Part struct {
	Text  string
	Style lipgloss.Style
}

// Plain is a row piece in the row's normal colour.
func Plain(text string) []Part { return []Part{{Text: text}} }

// Row draws one list row of the given width: left text, right text at the end,
// and a marker plus highlight when selected. Each part is drawn on top of
// the row's style, so the highlight also runs through coloured parts like ★.
func (t Theme) Row(left, right []Part, width int, selected, active bool) string {
	base, marker := t.Text, " "
	if selected {
		marker = t.Marker.Render(t.Glyph.Marker)
	}
	if selected && active {
		base = t.Selected
	}

	draw := func(parts []Part) (string, int) {
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
	return marker + l + gap + r + " "
}
