// Package tui is the terminal screen. app.go wires the components together.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/commands"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/commandbar"
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
	bar      commandbar.Model

	focus    pane
	agent    session.Source
	chosen   *session.Session
	terminal launch.Terminal      // nil when new tabs aren't supported
	paint    func(lipgloss.Color) // sets the terminal background; nil in tests

	width, height int

	saved   string   // theme to go back to when a preview ends
	help    []string // help text lines; nil when hidden
	helpTop int
}

func newModel(store *db.DB, sources []session.Source, home string, term launch.Terminal) model {
	current := func() string {
		name, _ := store.Setting("theme")
		if name == "" {
			return theme.Default
		}
		return name
	}
	t := theme.Get(current())
	svc := commands.New(command.Env{
		Store:   store,
		Themes:  theme.Names(),
		Current: current,
		Sources: sources,
		Home:    home,
		Keys:    keyLines(term != nil),
	})
	m := model{
		store:    store,
		sources:  sources,
		theme:    t,
		saved:    t.Name,
		agents:   agentlist.New(t),
		sessions: sessionlist.New(t, home),
		details:  details.New(t),
		status:   statusbar.New(t),
		bar:      commandbar.New(t, svc),
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
	m := newModel(store, sources, home, term)
	bg := newBackground()
	defer bg.restore() // before main resumes a session in this terminal
	m.paint = bg.set
	m.paint(m.theme.Background)

	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
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
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if k, ok := msg.(tea.KeyMsg); ok && m.bar.Opened() {
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(k)
		return m, cmd
	}

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

	case commandbar.RanMsg:
		m.setTheme(theme.Get(m.saved)) // drop any preview; a theme run sets it again
		return m, m.apply(msg.Event)

	case commandbar.PreviewMsg:
		if msg.Event == nil {
			m.setTheme(theme.Get(m.saved))
			return m, nil
		}
		return m, m.apply(msg.Event)

	case commandbar.ClosedMsg:
		m.setTheme(theme.Get(m.saved))
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.Command) {
			m.saved = m.theme.Name
			m.bar.Open()
			return m, nil
		}
		if m.help != nil {
			m.helpKey(msg)
			return m, nil
		}
		if m.focus == sessionPane && m.sessions.Searching() {
			break
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Refresh) && m.agent != nil:
			return m, m.apply(event.Refresh{})
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

// apply carries out what a command asked for.
func (m *model) apply(e event.Event) tea.Cmd {
	switch e := e.(type) {
	case event.ThemeChanged:
		m.setTheme(theme.Get(e.Name))
		if !e.Preview {
			m.saved = e.Name
			m.status.OK(fmt.Sprintf("theme %q", e.Name))
		}
	case event.Refresh:
		if m.agent == nil {
			return nil
		}
		m.status.Busy("refreshing…")
		return refresh(m.store, m.agent)
	case event.ShowHelp:
		m.help = strings.Split(strings.TrimSuffix(e.Text, "\n"), "\n")
		m.helpTop = 0
		m.status.SetHelp(helpViewHelp)
	case event.Quit:
		return tea.Quit
	default:
		m.status.Error(fmt.Errorf("unknown event %T", e))
	}
	return nil
}

// helpKey handles a key while help is shown; other keys are ignored.
func (m *model) helpKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc":
		m.help = nil
		m.setFocus(m.focus)
	case "up":
		m.helpTop = max(m.helpTop-1, 0)
	case "down":
		m.helpTop = min(m.helpTop+1, max(len(m.help)-m.bodyHeight(), 0))
	}
}

func (m *model) setTheme(t theme.Theme) {
	m.theme = t
	if m.paint != nil {
		m.paint(t.Background)
	}
	m.agents.SetTheme(t)
	m.sessions.SetTheme(t)
	m.details.SetTheme(t)
	m.status.SetTheme(t)
	m.bar.SetTheme(t)
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
