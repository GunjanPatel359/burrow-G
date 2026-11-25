// Package details is the box under the sessions list describing the selected session.
package details

import (
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

type Model struct {
	theme   theme.Theme
	width   int
	session *session.Session
	resume  []string
}

func New(t theme.Theme) Model { return Model{theme: t, width: 80} }

func (m *Model) SetWidth(w int)         { m.width = w }
func (m *Model) SetTheme(t theme.Theme) { m.theme = t }

func (m *Model) Show(s session.Session, resume []string) {
	m.session = &s
	m.resume = resume
}

func (m *Model) Clear() {
	m.session = nil
	m.resume = nil
}

func (m Model) View() string {
	if m.session == nil {
		return m.theme.Muted.Render(strings.Repeat(" ", theme.Gutter)+"no sessions") + strings.Repeat("\n", theme.DetailsHeight-1)
	}
	s := m.session
	t := m.theme
	pad := strings.Repeat(" ", theme.Gutter)
	fit := func(v string) string {
		return helpers.Truncate(v, m.width-theme.Gutter-theme.LabelWidth-1, t.Glyph.Ellipsis)
	}

	lines := []string{
		pad + t.Label.Render("id") + fit(s.ID),
		pad + t.Label.Render("folder") + t.Folder.Render(fit(s.Dir)),
		pad + t.Label.Render("resume") + fit(strings.Join(m.resume, " ")),
		pad + t.Label.Render("used") + helpers.TimeAgo(s.LastUsed, time.Now()) +
			t.Muted.Render(" "+t.Glyph.Sep+" "+s.LastUsed.Format("2006-01-02 15:04")),
	}
	return strings.Join(lines, "\n")
}
