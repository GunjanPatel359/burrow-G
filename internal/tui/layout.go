package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/details"
)

// Screen layout:
//
//	Agents │ Claude Code · 59
//	list   │ sessions list
//	       │ ──────────
//	       │ details
//	─────────────────────────
//	status line
const leftWidth = 20

func (m model) bodyHeight() int { return max(m.height-2, 8) }
func (m model) rightWidth() int { return max(m.width-leftWidth-1, 20) }
func (m model) listHeight() int { return max(m.bodyHeight()-2-details.Height, 1) }

func (m *model) resize() {
	m.agents.SetSize(leftWidth, m.bodyHeight()-1)
	m.sessions.SetSize(m.rightWidth(), m.listHeight())
	m.details.SetWidth(m.rightWidth())
}

func (m model) View() string {
	h := m.bodyHeight()
	t := m.theme

	left := t.Header.Render(" Agents") + "\n" + m.agents.View()
	left = lipgloss.NewStyle().Width(leftWidth).Height(h).MaxHeight(h).Render(left)

	sep := t.Muted.Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))

	list := lipgloss.NewStyle().Height(m.listHeight()).MaxHeight(m.listHeight()).Render(m.sessions.View())
	right := t.Header.Render(" "+m.sessionsTitle()) + "\n" +
		list + "\n" +
		t.Muted.Render(strings.Repeat("─", m.rightWidth())) + "\n" +
		m.details.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
	return body + "\n" + t.Muted.Render(strings.Repeat("─", m.width)) + "\n" + m.status.View()
}

func (m model) sessionsTitle() string {
	if m.agent == nil {
		return "No agents"
	}
	return fmt.Sprintf("%s · %d", agentlist.DisplayName(m.agent.Name()), m.sessions.Len())
}
