package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// Every key the app itself handles. Components handle their own (↑/↓, /).
var keys = struct {
	Quit, Enter, Here, Right, Left, Refresh, Bookmark, Command key.Binding
}{
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Enter:    key.NewBinding(key.WithKeys("enter")),
	Here:     key.NewBinding(key.WithKeys("o")),
	Right:    key.NewBinding(key.WithKeys("right", "tab", "l")),
	Left:     key.NewBinding(key.WithKeys("left", "shift+tab", "h", "esc")),
	Refresh:  key.NewBinding(key.WithKeys("r")),
	Bookmark: key.NewBinding(key.WithKeys("b")),
	Command:  key.NewBinding(key.WithKeys("ctrl+k")),
}

const agentHelp = "↑/↓ agent · enter/→ sessions · r refresh · ctrl+k commands · q quit"

// sessionHelp depends on whether new tabs are supported in this terminal.
func sessionHelp(tabs bool) string {
	if tabs {
		return "↑/↓ move · enter new tab · o resume here · b bookmark · / search · ← agents · r refresh · ctrl+k commands · q quit"
	}
	return "↑/↓ move · enter resume · b bookmark · / search · ← agents · r refresh · ctrl+k commands · q quit"
}

const helpViewHelp = "↑/↓ scroll · esc close · ctrl+k commands"

// keyLines is every key from the help lines, once each, for the help command.
func keyLines(tabs bool) []string {
	var out []string
	for _, k := range strings.Split(agentHelp+" · "+sessionHelp(tabs), " · ") {
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}
