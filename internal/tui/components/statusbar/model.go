// Package statusbar is the bottom line: a coloured status dot plus key help.
package statusbar

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

type Model struct {
	theme theme.Theme
	text  string
	style lipgloss.Style
	help  string
}

func New(t theme.Theme) Model { return Model{theme: t} }

func (m *Model) SetTheme(t theme.Theme) { m.theme = t }
func (m *Model) SetHelp(help string)    { m.help = help }

func (m *Model) Busy(text string) { m.text, m.style = text, m.theme.Busy }
func (m *Model) OK(text string)   { m.text, m.style = text, m.theme.OK }
func (m *Model) Error(err error)  { m.text, m.style = "error: "+err.Error(), m.theme.Error }
func (m *Model) Clear()           { m.text = "" }

func (m Model) View() string {
	dot := ""
	if m.text != "" {
		dot = m.style.Render("● "+m.text) + "   "
	}
	return " " + dot + m.theme.Muted.Render(m.help)
}
