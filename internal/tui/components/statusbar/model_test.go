package statusbar

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestView(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.SetHelp(key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")))

	m.Busy("refreshing…")
	if v := m.View(); !strings.Contains(v, "⠋ refreshing…") || !strings.Contains(v, "q quit") {
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

func TestSpinner(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{ASCII: true}))
	if m.Tick() != nil {
		t.Error("idle status bar should not tick")
	}

	first := m.Busy("refreshing...")
	if first == nil {
		t.Fatal("Busy should start the spinner")
	}
	if m.Busy("opening...") != nil {
		t.Error("Busy while spinning should not start a second spinner")
	}
	if !strings.Contains(m.View(), "| opening...") {
		t.Errorf("ASCII spinner should start on |: %q", m.View())
	}

	tick := first()
	m, next := m.Update(tick)
	if next == nil || !strings.Contains(m.View(), "/ opening...") {
		t.Errorf("a tick should move to the next frame and ask for another: %q", m.View())
	}

	m.OK("opened")
	if _, cmd := m.Update(tick); cmd != nil {
		t.Error("ticks after OK should be dropped")
	}

	// A tick from the spinner before this one is dropped too.
	m.Busy("refreshing...")
	if _, cmd := m.Update(tick.(spinner.TickMsg)); cmd != nil {
		t.Error("a tick from an earlier spinner should be dropped")
	}
}

func TestOKFades(t *testing.T) {
	FadeAfter = time.Millisecond
	m := New(theme.Get(theme.Default, theme.Env{}))
	cmd := m.OK("exported")
	if cmd == nil {
		t.Fatal("OK returned no fade command")
	}
	m, _ = m.Update(cmd())
	if strings.Contains(m.View(), "exported") {
		t.Errorf("view still shows the message after it faded: %q", m.View())
	}
}

func TestFadeKeepsNewerStatus(t *testing.T) {
	FadeAfter = time.Millisecond
	m := New(theme.Get(theme.Default, theme.Env{}))
	old := m.OK("refreshed")
	m.OK("bookmarked")
	m, _ = m.Update(old())
	if !strings.Contains(m.View(), "bookmarked") {
		t.Errorf("an older fade cleared a newer message: %q", m.View())
	}

	fade := m.OK("refreshed")
	m.Error(errors.New("disk full"))
	m, _ = m.Update(fade())
	if !strings.Contains(m.View(), "disk full") {
		t.Errorf("a fade cleared an error: %q", m.View())
	}
}
