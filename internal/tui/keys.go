package tui

import (
	"slices"

	"github.com/charmbracelet/bubbles/key"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// Every key the app itself handles. Help keys name arrows ("up", "left");
// ui.KeyName draws them from the glyph set.
var keys = struct {
	Quit, Enter, Here, Right, Left, Refresh, Bookmark, Command key.Binding
}{
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "resume")),
	Here:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "resume here")),
	Right:    key.NewBinding(key.WithKeys("right", "tab", "l"), key.WithHelp("right", "sessions")),
	Left:     key.NewBinding(key.WithKeys("left", "shift+tab", "h", "esc"), key.WithHelp("left", "agents")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Bookmark: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "bookmark")),
	Command:  key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "commands")),
}

// Keys the components handle, or the same keys with another word on another
// screen. They are only shown, never matched.
var hints = struct {
	Agent, Move, Scroll, Open, NewTab, Search, Close key.Binding
}{
	Agent:  key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "agent")),
	Move:   key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "move")),
	Scroll: key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "scroll")),
	Open:   key.NewBinding(key.WithKeys("enter", "right"), key.WithHelp("enter/right", "sessions")),
	NewTab: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new tab")),
	Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func agentHints() []key.Binding {
	return []key.Binding{hints.Agent, hints.Open, keys.Refresh, keys.Command, keys.Quit}
}

// sessionHints depends on whether new tabs are supported in this terminal.
func sessionHints(tabs bool) []key.Binding {
	open := []key.Binding{keys.Enter}
	if tabs {
		open = []key.Binding{hints.NewTab, keys.Here}
	}
	return slices.Concat([]key.Binding{hints.Move}, open,
		[]key.Binding{keys.Bookmark, hints.Search, keys.Left, keys.Refresh, keys.Command, keys.Quit})
}

func helpViewHints() []key.Binding {
	return []key.Binding{hints.Scroll, hints.Close, keys.Command}
}

// keyLines is every hint from both panes, once each, for the help command.
func keyLines(tabs bool, g theme.GlyphSet) []string {
	var out []string
	for _, b := range slices.Concat(agentHints(), sessionHints(tabs)) {
		if line := ui.HintText(g, b); !slices.Contains(out, line) {
			out = append(out, line)
		}
	}
	return out
}
