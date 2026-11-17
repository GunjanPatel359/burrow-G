// Package theme turns a colour palette into ready-to-use styles.
package theme

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name string

	Header   lipgloss.Style
	Muted    lipgloss.Style
	Text     lipgloss.Style
	Star     lipgloss.Style
	Folder   lipgloss.Style
	Label    lipgloss.Style
	Selected lipgloss.Style
	Marker   lipgloss.Style
	OK       lipgloss.Style
	Busy     lipgloss.Style
	Error    lipgloss.Style
}

// Get returns the named theme, or the default one if the name is unknown.
func Get(name string) Theme {
	p, ok := palettes[name]
	if !ok {
		name, p = Default, palettes[Default]
	}
	s := lipgloss.NewStyle
	return Theme{
		Name:     name,
		Header:   s().Foreground(p.Accent).Bold(true),
		Muted:    s().Foreground(p.Muted),
		Text:     s().Foreground(p.Text),
		Star:     s().Foreground(p.Warning),
		Folder:   s().Foreground(p.Info),
		Label:    s().Foreground(p.Label).Width(8),
		Selected: s().Background(p.Selected).Foreground(p.Text).Bold(true),
		Marker:   s().Foreground(p.Accent),
		OK:       s().Foreground(p.Success),
		Busy:     s().Foreground(p.Warning),
		Error:    s().Foreground(p.Error),
	}
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
// and a "▌" marker plus highlight when selected. Each part is drawn on top of
// the row's style, so the highlight also runs through coloured parts like ★.
func (t Theme) Row(left, right []Part, width int, selected, active bool) string {
	base, marker := t.Text, " "
	if selected {
		marker = t.Marker.Render("▌")
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
