// Package tui is the terminal screen. app.go wires the components together.
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/details"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/sessionlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/statusbar"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

type pane int

const (
	agentPane pane = iota
	sessionPane
)

type model struct {
	store   *db.DB
	sources []session.Source
	theme   theme.Theme

	agents   agentlist.Model
	sessions sessionlist.Model
	details  details.Model
	status   statusbar.Model

	focus    pane
	agent    session.Source
	chosen   *session.Session
	terminal launch.Terminal // nil when new tabs aren't supported

	width, height int
}

func newModel(store *db.DB, sources []session.Source, home string, term launch.Terminal) model {
	t := theme.Get(theme.Default)
	m := model{
		store:    store,
		sources:  sources,
		theme:    t,
		agents:   agentlist.New(t),
		sessions: sessionlist.New(t, home),
		details:  details.New(t),
		status:   statusbar.New(t),
		terminal: term,
		width:    100,
		height:   30,
	}
	m.loadAgents()
	if len(sources) > 0 {
		m.agent = sources[0]
		m.loadSessions()
		m.status.Busy("refreshing…")
	}
	m.setFocus(agentPane)
	m.resize()
	return m
}

// Run shows the TUI and returns the session to resume, or nil.
func Run(store *db.DB, sources []session.Source, home string, term launch.Terminal) (*session.Session, error) {
	final, err := tea.NewProgram(newModel(store, sources, home, term), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}
	return final.(model).chosen, nil
}

func (m model) Init() tea.Cmd {
	if m.agent == nil {
		return nil
	}
	return refresh(m.store, m.agent)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case refreshedMsg:
		m.loadAgents()
		if m.agent != nil && msg.agent == m.agent.Name() {
			m.loadSessions()
			if msg.err != nil {
				m.status.Error(msg.err)
			} else {
				m.status.OK("refreshed")
			}
		}
		return m, nil

	case openedMsg:
		if msg.err != nil {
			m.status.Error(msg.err)
		} else {
			m.status.OK(fmt.Sprintf("opened %q in a new tab", msg.title))
		}
		return m, nil

	case bookmarkedMsg:
		if msg.err != nil {
			m.status.Error(msg.err)
			return m, nil
		}
		if m.agent != nil && msg.s.Agent == m.agent.Name() {
			m.loadSessions()
			m.sessions.Select(msg.s.Key()) // it moved; keep the cursor on it
			m.syncDetails()
		}
		if msg.on {
			m.status.OK(fmt.Sprintf("bookmarked %q", msg.s.Title))
		} else {
			m.status.OK(fmt.Sprintf("removed bookmark from %q", msg.s.Title))
		}
		return m, nil

	case agentlist.SelectedMsg:
		return m, m.selectAgent(msg.Name)

	case tea.KeyMsg:
		if m.focus == sessionPane && m.sessions.Searching() {
			break
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Refresh) && m.agent != nil:
			m.status.Busy("refreshing…")
			return m, refresh(m.store, m.agent)
		case m.focus == agentPane && (key.Matches(msg, keys.Enter) || key.Matches(msg, keys.Right)):
			m.setFocus(sessionPane)
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Enter) && m.terminal != nil:
			if s, ok := m.sessions.Selected(); ok {
				m.status.Busy("opening…")
				return m, openTab(m.terminal, s, m.agent.ResumeCommand(s))
			}
			return m, nil
		case m.focus == sessionPane && (key.Matches(msg, keys.Enter) || key.Matches(msg, keys.Here)):
			if s, ok := m.sessions.Selected(); ok {
				m.chosen = &s
				return m, tea.Quit
			}
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Bookmark):
			if s, ok := m.sessions.Selected(); ok {
				return m, bookmark(m.store, s, !s.Bookmarked)
			}
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Left) && !m.sessions.Filtered():
			m.setFocus(agentPane)
			return m, nil
		case m.focus == agentPane && key.Matches(msg, keys.Left):
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.focus == agentPane {
		m.agents, cmd = m.agents.Update(msg)
	} else {
		m.sessions, cmd = m.sessions.Update(msg)
		m.syncDetails()
	}
	return m, cmd
}

func (m *model) selectAgent(name string) tea.Cmd {
	for _, src := range m.sources {
		if src.Name() == name {
			m.agent = src
		}
	}
	m.sessions.ResetCursor()
	m.loadSessions()
	m.status.Busy("refreshing…")
	return refresh(m.store, m.agent)
}

func (m *model) setFocus(p pane) {
	m.focus = p
	m.agents.SetFocused(p == agentPane)
	m.sessions.SetFocused(p == sessionPane)
	if p == agentPane {
		m.status.SetHelp(agentHelp)
	} else {
		m.status.SetHelp(sessionHelp(m.terminal != nil))
	}
}

func (m *model) loadAgents() {
	counts, err := m.store.Counts()
	if err != nil {
		m.status.Error(err)
	}
	var list []agentlist.Agent
	for _, src := range m.sources {
		list = append(list, agentlist.Agent{Name: src.Name(), Count: counts[src.Name()]})
	}
	m.agents.SetAgents(list)
}

func (m *model) loadSessions() {
	rows, err := m.store.List(m.agent.Name())
	if err != nil {
		m.status.Error(err)
	}
	m.sessions.SetSessions(rows)
	m.syncDetails()
}

func (m *model) syncDetails() {
	if s, ok := m.sessions.Selected(); ok {
		m.details.Show(s, m.agent.ResumeCommand(s))
	} else {
		m.details.Clear()
	}
}
