// Package statusbar is the bottom line: a colored status dot plus key help.
package statusbar

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

type Model struct {
	theme  theme.Theme
	status ui.Status
	help   []key.Binding
}

func New(t theme.Theme) Model { return Model{theme: t} }

func (m *Model) SetTheme(t theme.Theme)          { m.theme = t }
func (m *Model) SetHelp(bindings ...key.Binding) { m.help = bindings }

func (m *Model) Busy(text string) { m.status = ui.Status{Kind: ui.Busy, Text: text} }
func (m *Model) OK(text string)   { m.status = ui.Status{Kind: ui.OK, Text: text} }
func (m *Model) Error(err error)  { m.status = ui.Status{Kind: ui.Error, Text: "error: " + err.Error()} }
func (m *Model) Clear()           { m.status = ui.Status{} }

func (m Model) View() string {
	dot := ""
	if m.status.Text != "" {
		dot = ui.StatusDot(m.theme, m.status) + "   "
	}
	return " " + dot + ui.KeyHints(m.theme, m.help...)
}
