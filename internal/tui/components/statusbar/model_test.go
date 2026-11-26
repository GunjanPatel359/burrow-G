package statusbar

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/x/ansi"

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

func TestView_FitsWidth(t *testing.T) {
	bind := func(k, desc string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc)) }
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.SetHelp(bind("a", "first"), bind("b", "second"), bind("c", "third"))
	m.OK("done")

	m.SetWidth(35)
	v := ansi.Strip(m.View())
	if !strings.Contains(v, "a first · b second") || strings.Contains(v, "c third") || ansi.StringWidth(v) > 35 {
		t.Errorf("want the last hint dropped whole: %q", v)
	}

	m.OK(strings.Repeat("x", 50))
	if v := ansi.Strip(m.View()); ansi.StringWidth(v) > 35 || strings.Contains(v, "first") {
		t.Errorf("a long message keeps the line and is cut to fit: %q", v)
	}

	m.SetWidth(0)
	if v := ansi.Strip(m.View()); !strings.Contains(v, "c third") {
		t.Errorf("no width draws every hint: %q", v)
	}
}
