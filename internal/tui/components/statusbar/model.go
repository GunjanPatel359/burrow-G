// Package statusbar is the bottom line: a status dot (a spinner while busy)
// plus key help.
package statusbar

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// FadeAfter is how long a done message stays before it clears. Errors stay
// until something replaces them, so they can be read.
var FadeAfter = 2 * time.Second

// FadedMsg clears the done message that asked for it, if nothing has
// replaced it since.
type FadedMsg struct{ seq int }

type Model struct {
	theme  theme.Theme
	status ui.Status
	seq    int // counts status changes, so a late fade can't clear a newer one
	spin   spinner.Model
	help   []key.Binding
}

func New(t theme.Theme) Model { return Model{theme: t} }

func (m *Model) SetTheme(t theme.Theme)          { m.theme = t }
func (m *Model) SetHelp(bindings ...key.Binding) { m.help = bindings }

// Busy shows text with a spinner and returns its first tick. When it is
// already spinning only the text changes, so the spinner never runs twice.
func (m *Model) Busy(text string) tea.Cmd {
	spinning := m.busy()
	m.set(ui.Status{Kind: ui.Busy, Text: text})
	if spinning {
		return nil
	}
	// A new spinner gets a new ID, so ticks still on their way from an
	// earlier one are dropped.
	g := m.theme.Glyph.Spinner
	m.spin = spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: g.Frames,
		FPS:    time.Second / time.Duration(g.FPS),
	}))
	return m.spin.Tick
}

// OK shows a done message and returns the command that fades it.
func (m *Model) OK(text string) tea.Cmd {
	m.set(ui.Status{Kind: ui.OK, Text: text})
	seq := m.seq
	return tea.Tick(FadeAfter, func(time.Time) tea.Msg { return FadedMsg{seq: seq} })
}

func (m *Model) Error(err error) { m.set(ui.Status{Kind: ui.Error, Text: "error: " + err.Error()}) }
func (m *Model) Clear()          { m.set(ui.Status{}) }

func (m *Model) set(s ui.Status) {
	m.status = s
	m.seq++
}

// Tick starts the spinner when busy, for a Busy set before the program ran.
func (m Model) Tick() tea.Cmd {
	if !m.busy() {
		return nil
	}
	return m.spin.Tick
}

// Update clears a faded done message and moves the spinner on. Once the status is no longer busy its ticks are
// dropped, so nothing ticks while idle.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if f, ok := msg.(FadedMsg); ok {
		if f.seq == m.seq && m.status.Kind == ui.OK {
			m.Clear()
		}
		return m, nil
	}
	if !m.busy() {
		return m, nil
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m Model) busy() bool { return m.status.Kind == ui.Busy && m.status.Text != "" }

func (m Model) View() string {
	dot := ""
	if m.status.Text != "" {
		s := m.status
		if m.busy() {
			s.Frame = m.spin.View()
		}
		dot = ui.StatusDot(m.theme, s) + "   "
	}
	return " " + dot + ui.KeyHints(m.theme, m.help...)
}
