// Package commandbar is the command line: an input plus suggestion rows.
package commandbar

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// RanMsg: enter succeeded and the bar has closed.
type RanMsg struct{ Event event.Event }

// PreviewMsg: the selection moved; Event is the row's preview, nil if none.
type PreviewMsg struct{ Event event.Event }

// ClosedMsg: the bar closed without running anything.
type ClosedMsg struct{}

// maxRows is how many suggestions show at once; the rest scroll.
const maxRows = 6

type Model struct {
	theme    theme.Theme
	svc      *command.Service
	input    textinput.Model
	sugg     []command.Suggestion
	selected int
	top      int  // first suggestion shown
	moved    bool // the user picked a row with ↑/↓
	err      error
	width    int
	opened   bool
}

func New(t theme.Theme, svc *command.Service) Model {
	in := textinput.New()
	in.Prompt = "> "
	return Model{theme: t, svc: svc, input: in}
}

func (m *Model) Open() {
	m.opened = true
	m.input.SetValue("")
	m.input.Focus()
	m.err = nil
	m.refresh()
}

func (m Model) Opened() bool { return m.opened }

func (m *Model) SetTheme(t theme.Theme) { m.theme = t }

func (m *Model) SetWidth(w int) {
	m.width = w
	m.input.Width = max(w-3, 1)
}

func (m *Model) close() {
	m.opened = false
	m.input.Blur()
}

func (m *Model) refresh() {
	m.sugg = m.svc.Suggest(m.input.Value())
	m.selected = 0
	m.top = 0
	m.moved = false
}

// preview sends the highlighted row's preview, nil if none.
func (m Model) preview() tea.Cmd {
	var ev event.Event
	if len(m.sugg) > 0 {
		ev = m.sugg[m.selected].Preview
	}
	return func() tea.Msg { return PreviewMsg{Event: ev} }
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.opened {
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch key.String() {
	case "up", "down":
		if len(m.sugg) == 0 {
			return m, nil
		}
		if key.String() == "up" {
			m.selected = max(m.selected-1, 0)
		} else {
			m.selected = min(m.selected+1, len(m.sugg)-1)
		}
		m.top = min(m.top, m.selected)
		m.top = max(m.top, m.selected-maxRows+1)
		m.moved = true
		return m, m.preview()
	case "tab":
		if len(m.sugg) == 0 {
			return m, nil
		}
		ev := m.sugg[m.selected].Preview // show what was completed
		m.complete()
		return m, func() tea.Msg { return PreviewMsg{Event: ev} }
	case "enter":
		return m.enter()
	case "esc":
		if m.input.Value() != "" {
			m.input.SetValue("")
			m.err = nil
			m.refresh()
			return m, m.preview()
		}
		m.close()
		return m, func() tea.Msg { return ClosedMsg{} }
	case "ctrl+k":
		m.close()
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.err = nil
		m.refresh()
		cmd = tea.Batch(cmd, m.preview())
	}
	return m, cmd
}

// enter runs the line. A row picked with ↑/↓ goes in first; otherwise the
// line runs as typed, and if that fails the highlighted row is used, like tab.
// When the command still needs a value (e.g. "theme "), the bar stays open
// with the choices.
func (m Model) enter() (Model, tea.Cmd) {
	line := m.input.Value()
	var picked event.Event
	pick := func() {
		picked = m.sugg[m.selected].Preview
		line = m.svc.Complete(line, m.sugg[m.selected])
	}
	if m.moved && len(m.sugg) > 0 {
		pick()
	}
	ev, err := m.svc.Run(line)
	if err != nil && !m.moved && len(m.sugg) > 0 {
		pick()
		ev, err = m.svc.Run(line)
	}
	if err == nil {
		m.close()
		return m, func() tea.Msg { return RanMsg{Event: ev} }
	}

	m.input.SetValue(line)
	m.input.CursorEnd()
	m.refresh()
	if len(m.sugg) > 0 && strings.HasSuffix(line, " ") {
		m.err = nil // more to choose
		return m, func() tea.Msg { return PreviewMsg{Event: picked} }
	}
	m.err = err
	return m, nil
}

// complete puts the selected suggestion in the line.
func (m *Model) complete() {
	m.input.SetValue(m.svc.Complete(m.input.Value(), m.sugg[m.selected]))
	m.input.CursorEnd()
	m.refresh()
}

func (m Model) View() string {
	v := m.input.View()
	if m.err != nil {
		v += "\n" + m.theme.Error.Render(m.err.Error())
	}
	return v
}

func (m Model) Rows() []string {
	if len(m.sugg) == 0 {
		return nil
	}
	end := min(m.top+maxRows, len(m.sugg))
	rows := make([]string, 0, end-m.top)
	for i := m.top; i < end; i++ {
		s := m.sugg[i]
		textStyle := m.theme.Text
		if strings.HasPrefix(s.Text, "@") {
			textStyle = m.theme.Folder
		}
		left := []theme.Part{{Text: s.Text, Style: textStyle}}
		if s.Hint != "" {
			left = append(left, theme.Part{Text: "  " + s.Hint, Style: m.theme.Muted})
		}
		var right []theme.Part
		if s.Key != "" {
			right = []theme.Part{{Text: s.Key, Style: m.theme.Muted}}
		}
		rows = append(rows, ui.SurfaceRow(m.theme, left, right, m.width, i == m.selected))
	}
	return rows
}
