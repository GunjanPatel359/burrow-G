package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
	"github.com/GunjanPatel359/burrow-G/internal/tui/ui"
)

// Screen layout:
//
//	Agents │ Claude Code · 59
//	list   │ sessions list
//	       │ ──────────
//	       │ details
//	─────────────────────────
//	status line
//
// Below NarrowWidth only the focused pane shows, at full width; below
// ShortHeight the details box is hidden; below MinWidth × MinHeight only a
// note to make the window bigger shows.
func (m model) bodyHeight() int { return max(m.height-2, 1) }
func (m model) narrow() bool    { return m.width < theme.NarrowWidth }
func (m model) short() bool     { return m.height < theme.ShortHeight }
func (m model) tooSmall() bool  { return m.width < theme.MinWidth || m.height < theme.MinHeight }

func (m model) leftWidth() int {
	if m.narrow() {
		return m.width
	}
	return theme.AgentsWidth
}

func (m model) rightWidth() int {
	if m.narrow() {
		return m.width
	}
	return max(m.width-theme.AgentsWidth-1, 20)
}

func (m model) listHeight() int {
	if m.short() {
		return max(m.bodyHeight()-1, 1)
	}
	return max(m.bodyHeight()-2-theme.DetailsHeight, 1)
}

func (m *model) resize() {
	m.agents.SetSize(m.leftWidth(), m.bodyHeight()-1)
	m.sessions.SetSize(m.rightWidth(), m.listHeight())
	m.details.SetWidth(m.rightWidth())
	m.bar.SetWidth(m.width)
	m.status.SetWidth(m.width)
	m.helpTop = min(m.helpTop, max(len(m.help)-m.bodyHeight(), 0))
}

func (m model) View() string {
	t := m.theme
	if m.tooSmall() {
		need := fmt.Sprintf("needs %d%s%d", theme.MinWidth, t.Glyph.Times, theme.MinHeight)
		return ui.EmptyState(t, "make the window bigger", need, m.width, m.height)
	}
	h := m.bodyHeight()

	var body string
	switch {
	case m.narrow() && m.focus == agentPane && m.help == nil:
		body = m.leftView()
	case m.narrow():
		body = m.rightView()
	default:
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.leftView(), ui.VRule(t, h), m.rightView())
	}
	rows := strings.Split(body, "\n")
	rows = rows[:min(len(rows), h)]

	bottom := m.status.View()
	if m.bar.Opened() {
		// The bar may take more than one line; the body gives them up.
		bottom = m.bar.View()
		rows = rows[:max(len(rows)-strings.Count(bottom, "\n"), 0)]
		sugg := m.bar.Rows()
		sugg = sugg[:min(len(sugg), len(rows))]
		copy(rows[len(rows)-len(sugg):], sugg)
	}
	return strings.Join(rows, "\n") + "\n" + ui.Divider(t, m.width) + "\n" + bottom
}

func (m model) leftView() string {
	h := m.bodyHeight()
	left := ui.PaneHeader(m.theme, "Agents", m.focus == agentPane) + "\n" + m.agents.View()
	return lipgloss.NewStyle().Width(m.leftWidth()).Height(h).MaxHeight(h).Render(left)
}

// rightView is the sessions list and details, or the help text when shown.
func (m model) rightView() string {
	h := m.bodyHeight()
	t := m.theme
	if m.help != nil {
		pad := strings.Repeat(" ", theme.Gutter)
		text := t.Text.Render(pad + strings.Join(m.help[m.helpTop:], "\n"+pad))
		text = lipgloss.NewStyle().MaxWidth(m.rightWidth()).Render(text) // cut, don't wrap
		return lipgloss.NewStyle().Width(m.rightWidth()).Height(h).MaxHeight(h).Render(text)
	}
	out := ui.PaneHeader(t, m.sessionsTitle(), m.focus == sessionPane) + "\n" + m.listView()
	if !m.short() {
		out += "\n" + ui.Divider(t, m.rightWidth()) + "\n" + m.details.View()
	}
	return out
}

// listView is the sessions list, or why it is empty.
func (m model) listView() string {
	w, h := m.rightWidth(), m.listHeight()
	t := m.theme
	switch {
	case m.agent == nil:
		where := "burrow-g reads ~/.claude, ~/.qoder, ~/.codex, " + t.Glyph.Ellipsis
		return ui.EmptyState(t, "no agents found", where, w, h)
	case m.sessions.Len() == 0:
		name := agentlist.DisplayName(m.agent.Name())
		return ui.EmptyState(t, "no sessions yet", "start one in "+name+" and press r", w, h)
	}
	return lipgloss.NewStyle().Height(h).MaxHeight(h).Render(m.sessions.View())
}

func (m model) sessionsTitle() string {
	if m.agent == nil {
		return "No agents"
	}
	return fmt.Sprintf("%s %s %d", agentlist.DisplayName(m.agent.Name()), m.theme.Glyph.Sep, m.sessions.Len())
}
