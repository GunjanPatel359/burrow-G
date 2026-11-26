package agentpicker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	oKey  = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
)

func open(tabs bool) Model {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.Open("Fix it", []string{"claude", "codex"}, tabs)
	return m
}

func send(m Model, k tea.KeyMsg) (Model, tea.Msg) {
	m, cmd := m.Update(k)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

func TestPick(t *testing.T) {
	tests := []struct {
		name string
		tabs bool
		keys []tea.KeyMsg
		want tea.Msg
	}{
		{"enter opens a tab", true, []tea.KeyMsg{down, enter}, PickedMsg{Agent: "codex"}},
		{"enter without tabs runs here", false, []tea.KeyMsg{enter}, PickedMsg{Agent: "claude", Here: true}},
		{"o runs here", true, []tea.KeyMsg{oKey}, PickedMsg{Agent: "claude", Here: true}},
		{"esc cancels", true, []tea.KeyMsg{down, esc}, ClosedMsg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := open(tt.tabs)
			var got tea.Msg
			for _, k := range tt.keys {
				m, got = send(m, k)
			}
			if got != tt.want || m.Opened() {
				t.Errorf("got %#v (open %v), want %#v", got, m.Opened(), tt.want)
			}
		})
	}
}

func TestCursorStaysInRange(t *testing.T) {
	m := open(true)
	for range 5 {
		m, _ = send(m, down)
	}
	if _, got := send(m, enter); got != (PickedMsg{Agent: "codex"}) {
		t.Errorf("got %#v", got)
	}
}

func TestView(t *testing.T) {
	v := open(true).View(80)
	for _, want := range []string{"Hand off session", "Fix it", "Claude Code", "Codex", "new tab", "cancel"} {
		if !strings.Contains(v, want) {
			t.Errorf("view misses %q:\n%s", want, v)
		}
	}
}

func TestOpen_NoAgentsStaysClosed(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.Open("x", nil, true)
	if m.Opened() {
		t.Error("an empty picker must not open")
	}
}
