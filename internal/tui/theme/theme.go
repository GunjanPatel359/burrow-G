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

// Row draws one list row of the given width: left text, right text at the end,
// and a "▌" marker plus highlight when selected.
func (t Theme) Row(left, right string, width int, selected, active bool) string {
	gap := max(width-2-lipgloss.Width(left)-lipgloss.Width(right), 1)
	row := left + strings.Repeat(" ", gap) + right

	switch {
	case selected && active:
		return t.Marker.Render("▌") + t.Selected.Render(row) + " "
	case selected:
		return t.Marker.Render("▌") + t.Text.Render(row) + " "
	default:
		return " " + t.Text.Render(row) + " "
	}
}
