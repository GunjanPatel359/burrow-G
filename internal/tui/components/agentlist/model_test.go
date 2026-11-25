package agentlist

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func newTest() Model {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.SetSize(20, 10)
	m.SetAgents([]Agent{{Name: "claude", Count: 59}, {Name: "qoder", Count: 7}})
	return m
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("claude"); got != "Claude Code" {
		t.Errorf("DisplayName(claude) = %q", got)
	}
	if got := DisplayName("unknown-cli"); got != "unknown-cli" {
		t.Errorf("DisplayName(unknown) = %q", got)
	}
}

func TestView(t *testing.T) {
	v := newTest().View()
	for _, want := range []string{"Claude Code", "59", "Qoder", "7"} {
		if !strings.Contains(v, want) {
			t.Errorf("view is missing %q:\n%s", want, v)
		}
	}
}

func TestMovingSendsSelectedMsg(t *testing.T) {
	m := newTest()
	if m.Selected() != "claude" {
		t.Fatalf("Selected() = %q, want claude", m.Selected())
	}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Selected() != "qoder" {
		t.Fatalf("after ↓ Selected() = %q, want qoder", m.Selected())
	}
	if cmd == nil {
		t.Fatal("moving to another agent should send a message")
	}
	if msg, ok := cmd().(SelectedMsg); !ok || msg.Name != "qoder" {
		t.Errorf("got %#v, want SelectedMsg{qoder}", cmd())
	}
}

func TestSetAgentsKeepsSelection(t *testing.T) {
	m := newTest()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.SetAgents([]Agent{{Name: "claude", Count: 60}, {Name: "qoder", Count: 8}})
	if m.Selected() != "qoder" {
		t.Errorf("refreshing counts moved the cursor to %q", m.Selected())
	}
}
