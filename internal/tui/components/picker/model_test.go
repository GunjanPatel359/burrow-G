package picker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var (
	up    = tea.KeyMsg{Type: tea.KeyUp}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyO  = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
)

var items = []Item{{Label: "Save to file"}, {Label: "Continue in Codex", Agent: "codex"}}

func opened() Model {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.Open("Move session", `"Fix it" · Claude Code`, items, true)
	return m
}

func chose(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no message")
	}
	return cmd()
}

func TestEnterChoosesTheSelectedItem(t *testing.T) {
	m := opened()
	m, _ = m.Update(down)
	m, cmd := m.Update(enter)
	if got := chose(t, cmd); got != (ChoseMsg{Item: items[1], Here: false}) {
		t.Errorf("got %+v", got)
	}
	if m.Opened() {
		t.Error("the picker should close once something is chosen")
	}
}

func TestOChoosesHere(t *testing.T) {
	m := opened()
	m, _ = m.Update(down)
	_, cmd := m.Update(keyO)
	if got := chose(t, cmd); got != (ChoseMsg{Item: items[1], Here: true}) {
		t.Errorf("got %+v", got)
	}
}

func TestEscCloses(t *testing.T) {
	m, cmd := opened().Update(esc)
	if got := chose(t, cmd); got != (ClosedMsg{}) || m.Opened() {
		t.Errorf("got %+v, opened %v", got, m.Opened())
	}
}

func TestCursorStaysInside(t *testing.T) {
	m := opened()
	m, _ = m.Update(up)
	m, _ = m.Update(down)
	m, _ = m.Update(down)
	_, cmd := m.Update(enter)
	if got := chose(t, cmd).(ChoseMsg); got.Item != items[1] {
		t.Errorf("chose %+v, want the last item", got.Item)
	}
}

func TestViewShowsItemsAndKeys(t *testing.T) {
	v := opened().View(80)
	for _, want := range []string{"Move session", "Save to file", "Continue in Codex", "esc"} {
		if !strings.Contains(v, want) {
			t.Errorf("view is missing %q:\n%s", want, v)
		}
	}
}
