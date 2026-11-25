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
func (m model) bodyHeight() int { return max(m.height-2, 8) }
func (m model) rightWidth() int { return max(m.width-theme.AgentsWidth-1, 20) }
func (m model) listHeight() int { return max(m.bodyHeight()-2-theme.DetailsHeight, 1) }

func (m *model) resize() {
	m.agents.SetSize(theme.AgentsWidth, m.bodyHeight()-1)
	m.sessions.SetSize(m.rightWidth(), m.listHeight())
	m.details.SetWidth(m.rightWidth())
	m.bar.SetWidth(m.width)
	m.helpTop = min(m.helpTop, max(len(m.help)-m.bodyHeight(), 0))
}

func (m model) View() string {
	h := m.bodyHeight()
	t := m.theme

	left := ui.PaneHeader(t, "Agents", m.focus == agentPane) + "\n" + m.agents.View()
	left = lipgloss.NewStyle().Width(theme.AgentsWidth).Height(h).MaxHeight(h).Render(left)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, ui.VRule(t, h), m.rightView())
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
	list := lipgloss.NewStyle().Height(m.listHeight()).MaxHeight(m.listHeight()).Render(m.sessions.View())
	return ui.PaneHeader(t, m.sessionsTitle(), m.focus == sessionPane) + "\n" +
		list + "\n" +
		ui.Divider(t, m.rightWidth()) + "\n" +
		m.details.View()
}

func (m model) sessionsTitle() string {
	if m.agent == nil {
		return "No agents"
	}
	return fmt.Sprintf("%s %s %d", agentlist.DisplayName(m.agent.Name()), m.theme.Glyph.Sep, m.sessions.Len())
}
