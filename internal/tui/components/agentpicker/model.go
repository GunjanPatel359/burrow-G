// Package agentpicker is the dialog that picks which agent a session is
// handed off to.
package agentpicker

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// PickedMsg is sent when an agent is chosen. Here means this terminal, not a new tab.
type PickedMsg struct {
	Agent string
	Here  bool
}

// ClosedMsg is sent when the picker is cancelled.
type ClosedMsg struct{}

var keys = struct {
	Up, Down, Pick, PickTab, Here, Cancel key.Binding
}{
	Up:      key.NewBinding(key.WithKeys("up", "k")),
	Down:    key.NewBinding(key.WithKeys("down", "j")),
	Pick:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "hand off")),
	PickTab: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new tab")),
	Here:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "here")),
	Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

type Model struct {
	theme  theme.Theme
	agents []string
	cursor int
	open   bool
	tabs   bool // enter opens a new tab
	title  string
}

func New(t theme.Theme) Model { return Model{theme: t} }

// Open shows the picker for a session titled title.
func (m *Model) Open(title string, agents []string, tabs bool) {
	m.title, m.agents, m.tabs = title, agents, tabs
	m.cursor = 0
	m.open = len(agents) > 0
}

func (m Model) Opened() bool { return m.open }

func (m *Model) SetTheme(t theme.Theme) { m.theme = t }

func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(msg, keys.Down):
		m.cursor = min(m.cursor+1, len(m.agents)-1)
	case key.Matches(msg, keys.Pick):
		return m.pick(!m.tabs)
	case key.Matches(msg, keys.Here):
		return m.pick(true)
	case key.Matches(msg, keys.Cancel):
		m.open = false
		return m, func() tea.Msg { return ClosedMsg{} }
	}
	return m, nil
}

func (m Model) pick(here bool) (Model, tea.Cmd) {
	m.open = false
	picked := PickedMsg{Agent: m.agents[m.cursor], Here: here}
	return m, func() tea.Msg { return picked }
}

// View is the dialog box, sized for an area width cells wide.
func (m Model) View(width int) string {
	g := m.theme.Glyph
	body := []string{"Continue \"" + m.title + "\" in:", ""}
	for i, a := range m.agents {
		mark := " "
		if i == m.cursor {
			mark = g.Marker
		}
		body = append(body, mark+" "+agentlist.DisplayName(a))
	}
	hints := []key.Binding{keys.Pick, keys.Cancel}
	if m.tabs {
		hints = []key.Binding{keys.PickTab, keys.Here, keys.Cancel}
	}
	return ui.Dialog(m.theme, "Hand off session", body, hints, width)
}
