// Package picker is the dialog `e` opens: where a session goes, to a file or
// into another agent (spec 0004). While open it takes every key.
package picker

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// Item is one choice. An empty Agent means "save to file".
type Item struct {
	Label string
	Agent string
}

// ChoseMsg is sent when an item is chosen; Here means in this terminal (o)
// rather than the default (enter).
type ChoseMsg struct {
	Item Item
	Here bool
}

// ClosedMsg is sent when the picker closes with nothing chosen.
type ClosedMsg struct{}

var keys = struct {
	Up, Down, Choose, Here, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Choose: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	Here:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "here")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

var move = key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "move"))

type Model struct {
	theme  theme.Theme
	title  string
	lead   string
	items  []Item
	cursor int
	tabs   bool // offer "o here": enter opens a new tab
	opened bool
}

func New(t theme.Theme) Model { return Model{theme: t} }

func (m *Model) SetTheme(t theme.Theme) { m.theme = t }

// Open shows items under title and a lead line. tabs says whether enter opens
// a new tab, so o (here) is worth offering.
func (m *Model) Open(title, lead string, items []Item, tabs bool) {
	*m = Model{theme: m.theme, title: title, lead: lead, items: items, tabs: tabs, opened: true}
}

func (m Model) Opened() bool { return m.opened }

func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(msg, keys.Down):
		m.cursor = min(m.cursor+1, len(m.items)-1)
	case key.Matches(msg, keys.Choose), key.Matches(msg, keys.Here):
		chosen := ChoseMsg{Item: m.items[m.cursor], Here: key.Matches(msg, keys.Here)}
		m.opened = false
		return m, func() tea.Msg { return chosen }
	case key.Matches(msg, keys.Close):
		m.opened = false
		return m, func() tea.Msg { return ClosedMsg{} }
	}
	return m, nil
}

// View is the dialog for a screen width wide.
func (m Model) View(width int) string {
	labels := make([]string, len(m.items))
	for i, it := range m.items {
		labels[i] = it.Label
	}
	hints := []key.Binding{move, keys.Choose, keys.Close}
	if m.tabs {
		hints = []key.Binding{move, keys.Choose, keys.Here, keys.Close}
	}
	return ui.Choice(m.theme, m.title, []string{m.lead}, labels, m.cursor, hints, width)
}
