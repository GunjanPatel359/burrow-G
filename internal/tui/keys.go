package tui

import "github.com/charmbracelet/bubbles/key"

// Every key the app itself handles. Components handle their own (↑/↓, /).
var keys = struct {
	Quit, Enter, Here, Right, Left, Refresh key.Binding
}{
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Enter:   key.NewBinding(key.WithKeys("enter")),
	Here:    key.NewBinding(key.WithKeys("o")),
	Right:   key.NewBinding(key.WithKeys("right", "tab", "l")),
	Left:    key.NewBinding(key.WithKeys("left", "shift+tab", "h", "esc")),
	Refresh: key.NewBinding(key.WithKeys("r")),
}

const agentHelp = "↑/↓ agent · enter/→ sessions · r refresh · q quit"

// sessionHelp depends on whether new tabs are supported in this terminal.
func sessionHelp(tabs bool) string {
	if tabs {
		return "↑/↓ move · enter new tab · o resume here · / search · ← agents · r refresh · q quit"
	}
	return "↑/↓ move · enter resume · / search · ← agents · r refresh · q quit"
}
