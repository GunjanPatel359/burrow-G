// Package agentlist is the left pane: one row per agent with its session count.
package agentlist

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// SelectedMsg is sent when the cursor moves to another agent.
type SelectedMsg struct{ Name string }

type Agent struct {
	Name  string
	Count int
}

func (a Agent) FilterValue() string { return a.Name }

var displayNames = map[string]string{
	"claude":   "Claude Code",
	"qoder":    "Qoder",
	"opencode": "OpenCode",
	"copilot":  "Copilot CLI",
	"codex":    "Codex",
}

func DisplayName(agent string) string {
	if name := displayNames[agent]; name != "" {
		return name
	}
	return agent
}

type Model struct {
	list    list.Model
	theme   theme.Theme
	focused bool
}

func New(t theme.Theme) Model {
	l := list.New(nil, delegate{theme: t}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	return Model{list: l, theme: t}
}

func (m *Model) SetAgents(agents []Agent) {
	items := make([]list.Item, len(agents))
	for i, a := range agents {
		items[i] = a
	}
	m.list.SetItems(items)
}

func (m *Model) SetSize(w, h int) { m.list.SetSize(w, h) }

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	m.list.SetDelegate(delegate{theme: m.theme, active: m.focused})
}

func (m *Model) SetTheme(t theme.Theme) {
	m.theme = t
	m.list.SetDelegate(delegate{theme: m.theme, active: m.focused})
}

// Selected is the name of the agent under the cursor.
func (m Model) Selected() string {
	if a, ok := m.list.SelectedItem().(Agent); ok {
		return a.Name
	}
	return ""
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	before := m.Selected()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	if now := m.Selected(); now != before {
		selected := func() tea.Msg { return SelectedMsg{Name: now} }
		return m, tea.Batch(cmd, selected)
	}
	return m, cmd
}

func (m Model) View() string { return m.list.View() }

// delegate draws one row: "Claude Code ......... 59".
type delegate struct {
	theme  theme.Theme
	active bool
}

func (delegate) Height() int                         { return 1 }
func (delegate) Spacing() int                        { return 0 }
func (delegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	a := item.(Agent)
	count := fmt.Sprint(a.Count)
	name := helpers.Truncate(DisplayName(a.Name), m.Width()-len(count)-3, d.theme.Glyph.Ellipsis)
	fmt.Fprint(w, ui.Row(d.theme, theme.Plain(name), theme.Plain(count), m.Width(), index == m.Index(), d.active))
}
