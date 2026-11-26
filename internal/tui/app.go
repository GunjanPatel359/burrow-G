// Package tui is the terminal screen. app.go wires the components together.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/commands"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentpicker"
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

// Launch is what to run in this terminal once the UI exits: a resume or a handoff.
type Launch struct {
	Session session.Session
	Argv    []string
	Report  string // printed before the terminal is taken over; "" for none
}

// selection is shared by every copy of the model, so the command bar sees
// the session highlighted now, not when it was built.
type selection struct {
	s  session.Session
	ok bool
}

type model struct {
	store    *db.DB
	sources  []session.Source
	home     string
	lookPath func(string) (string, error)
	theme    theme.Theme
	env      theme.Env // what the terminal can draw; every theme.Get uses it

	agents   agentlist.Model
	sessions sessionlist.Model
	details  details.Model
	status   statusbar.Model
	bar      commandbar.Model
	picker   agentpicker.Model
	pickFor  session.Session // the session the picker hands off
	sel      *selection

	focus    pane
	agent    session.Source
	chosen   *Launch
	terminal launch.Terminal      // nil when new tabs aren't supported
	paint    func(lipgloss.Color) // sets the terminal background; nil in tests

	width, height int

	saved   string   // theme to go back to when a preview ends
	help    []string // help text lines; nil when hidden
	helpTop int
}

// newModel builds the app. dark reports the terminal's background and picks
// the first theme when none is saved.
func newModel(store *db.DB, sources []session.Source, home string, term launch.Terminal, env theme.Env, dark func() bool) model {
	current := func() string {
		name, _ := store.Setting("theme")
		return theme.Initial(name, dark)
	}
	t := theme.Get(current(), env)
	sel := &selection{}
	svc := commands.New(command.Env{
		Store:   store,
		Themes:  theme.Names(),
		Current: current,
		Sources: sources,
		Home:    home,
		Keys:    keyLines(term != nil, t.Glyph),
		Selected: func() (session.Session, bool) {
			return sel.s, sel.ok
		},
		LookPath: exec.LookPath,
	})
	m := model{
		store:    store,
		sources:  sources,
		home:     home,
		lookPath: exec.LookPath,
		sel:      sel,
		picker:   agentpicker.New(t),
		theme:    t,
		env:      env,
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
		m.status.Busy("refreshing" + m.theme.Glyph.Ellipsis) // Init starts its spinner
	}
	m.setFocus(agentPane)
	m.resize()
	return m
}

// Run shows the TUI and returns what to run in this terminal, or nil.
func Run(store *db.DB, sources []session.Source, home string, term launch.Terminal) (*Launch, error) {
	env := theme.DetectEnv(lipgloss.DefaultRenderer().Output().EnvColorProfile(), os.Getenv)
	lipgloss.SetColorProfile(env.Depth.Profile())
	bg := newBackground()
	defer bg.restore() // before main resumes a session in this terminal
	m := newModel(store, sources, home, term, env, bg.dark)
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
	return tea.Batch(refresh(m.store, m.agent), m.status.Tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if k, ok := msg.(tea.KeyMsg); ok && m.picker.Opened() {
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(k)
		return m, cmd
	}
	if k, ok := msg.(tea.KeyMsg); ok && m.bar.Opened() {
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(k)
		return m, cmd
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.status, cmd = m.status.Update(msg)
		return m, cmd

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
		switch {
		case msg.err != nil:
			m.status.Error(msg.err)
		case msg.report != "":
			m.status.OK(msg.report)
		default:
			m.status.OK(fmt.Sprintf("opened %q in a new tab", msg.title))
		}
		return m, nil

	case exportedMsg:
		if msg.err != nil {
			m.status.Error(msg.err)
		} else {
			m.status.OK(transcript.ExportReport(msg.stats) + " to " + helpers.ShortPath(msg.path, m.home))
		}
		return m, nil

	case handoffReadyMsg:
		if msg.err != nil {
			m.status.Error(msg.err)
			return m, nil
		}
		report := transcript.Report(msg.agent, msg.stats)
		if msg.here || m.terminal == nil {
			m.chosen = &Launch{Session: msg.s, Argv: msg.argv, Report: report}
			return m, tea.Quit
		}
		busy := m.status.Busy("opening" + m.theme.Glyph.Ellipsis)
		return m, tea.Batch(busy, openTab(m.terminal, msg.s, msg.argv, report))

	case agentpicker.PickedMsg:
		return m, m.handoff(m.pickFor, msg.Agent, msg.Here)

	case agentpicker.ClosedMsg:
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
		m.setTheme(theme.Get(m.saved, m.env)) // drop any preview; a theme run sets it again
		return m, m.apply(msg.Event)

	case commandbar.PreviewMsg:
		if msg.Event == nil {
			m.setTheme(theme.Get(m.saved, m.env))
			return m, nil
		}
		return m, m.apply(msg.Event)

	case commandbar.ClosedMsg:
		m.setTheme(theme.Get(m.saved, m.env))
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
				busy := m.status.Busy("opening" + m.theme.Glyph.Ellipsis)
				return m, tea.Batch(busy, openTab(m.terminal, s, m.agent.ResumeCommand(s), ""))
			}
			return m, nil
		case m.focus == sessionPane && (key.Matches(msg, keys.Enter) || key.Matches(msg, keys.Here)):
			if s, ok := m.sessions.Selected(); ok {
				m.chosen = &Launch{Session: s, Argv: m.agent.ResumeCommand(s)}
				return m, tea.Quit
			}
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Bookmark):
			if s, ok := m.sessions.Selected(); ok {
				return m, bookmark(m.store, s, !s.Bookmarked)
			}
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Export):
			if s, ok := m.sessions.Selected(); ok {
				return m, m.apply(event.ExportRequested{Session: s})
			}
			return m, nil
		case m.focus == sessionPane && key.Matches(msg, keys.Handoff):
			if s, ok := m.sessions.Selected(); ok {
				return m, m.apply(event.HandoffRequested{Session: s})
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
		m.setTheme(theme.Get(e.Name, m.env))
		if !e.Preview {
			m.saved = e.Name
			m.status.OK(themeSaved(e.Name, m.env))
		}
	case event.Refresh:
		if m.agent == nil {
			return nil
		}
		busy := m.status.Busy("refreshing" + m.theme.Glyph.Ellipsis)
		return tea.Batch(busy, refresh(m.store, m.agent))
	case event.ShowHelp:
		m.help = strings.Split(strings.TrimSuffix(e.Text, "\n"), "\n")
		m.helpTop = 0
		m.status.SetHelp(helpViewHints()...)
	case event.ExportRequested:
		src, err := m.transcriber(e.Session.Agent)
		if err != nil {
			m.status.Error(err)
			return nil
		}
		busy := m.status.Busy("exporting" + m.theme.Glyph.Ellipsis)
		return tea.Batch(busy, exportSession(src, e.Session, m.home))
	case event.HandoffRequested:
		if e.Agent != "" {
			return m.handoff(e.Session, e.Agent, false)
		}
		var names []string
		for _, src := range transcript.Targets(m.sources, m.lookPath) {
			names = append(names, src.Name())
		}
		if len(names) == 0 {
			m.status.Error(fmt.Errorf("no agent CLI found on PATH"))
			return nil
		}
		m.pickFor = e.Session
		m.picker.Open(e.Session.Title, names, m.terminal != nil)
	case event.Quit:
		return tea.Quit
	default:
		m.status.Error(fmt.Errorf("unknown event %T", e))
	}
	return nil
}

// handoff builds the prompt in the background; handoffReadyMsg launches it.
func (m *model) handoff(s session.Session, agent string, here bool) tea.Cmd {
	src, err := m.transcriber(s.Agent)
	if err != nil {
		m.status.Error(err)
		return nil
	}
	dst, err := m.transcriber(agent)
	if err != nil {
		m.status.Error(err)
		return nil
	}
	busy := m.status.Busy("preparing handoff" + m.theme.Glyph.Ellipsis)
	return tea.Batch(busy, prepareHandoff(src, dst, s, agent, here, m.lookPath))
}

func (m *model) transcriber(agent string) (session.Transcriber, error) {
	for _, src := range m.sources {
		if src.Name() != agent {
			continue
		}
		if tr, ok := src.(session.Transcriber); ok {
			return tr, nil
		}
		return nil, fmt.Errorf("export and handoff not supported for %s", agent)
	}
	return nil, fmt.Errorf("unknown agent %q", agent)
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
	m.picker.SetTheme(t)
}

func (m *model) selectAgent(name string) tea.Cmd {
	for _, src := range m.sources {
		if src.Name() == name {
			m.agent = src
		}
	}
	m.sessions.ResetCursor()
	m.loadSessions()
	busy := m.status.Busy("refreshing" + m.theme.Glyph.Ellipsis)
	return tea.Batch(busy, refresh(m.store, m.agent))
}

// themeSaved reports a saved theme. Without truecolor or 256 colors every
// theme looks the same, so it says why nothing changed.
func themeSaved(name string, env theme.Env) string {
	if !env.Depth.Paints() {
		return fmt.Sprintf("theme %q (colors need truecolor or 256)", name)
	}
	return fmt.Sprintf("theme %q", name)
}

func (m *model) setFocus(p pane) {
	m.focus = p
	m.agents.SetFocused(p == agentPane)
	m.sessions.SetFocused(p == sessionPane)
	if p == agentPane {
		m.status.SetHelp(agentHints()...)
	} else {
		m.status.SetHelp(sessionHints(m.terminal != nil)...)
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
	s, ok := m.sessions.Selected()
	m.sel.s, m.sel.ok = s, ok
	if ok {
		m.details.Show(s, m.agent.ResumeCommand(s))
	} else {
		m.details.Clear()
	}
}
