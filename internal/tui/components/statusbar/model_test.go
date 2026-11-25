package statusbar

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestView(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.SetHelp(key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")))

	m.Busy("refreshing…")
	if v := m.View(); !strings.Contains(v, "● refreshing…") || !strings.Contains(v, "q quit") {
		t.Errorf("busy view = %q", v)
	}

	m.OK("refreshed")
	if v := m.View(); !strings.Contains(v, "● refreshed") {
		t.Errorf("ok view = %q", v)
	}

	m.Error(errors.New("db locked"))
	if v := m.View(); !strings.Contains(v, "● error: db locked") {
		t.Errorf("error view = %q", v)
	}

	m.Clear()
	if v := m.View(); strings.Contains(v, "●") {
		t.Errorf("cleared view should have no dot: %q", v)
	}
}
