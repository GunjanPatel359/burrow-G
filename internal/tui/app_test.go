package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
)

type fakeSource struct {
	name string
	list []session.Session
}

func (f fakeSource) Name() string                     { return f.name }
func (f fakeSource) List() ([]session.Session, error) { return f.list, nil }
func (f fakeSource) ResumeCommand(s session.Session) []string {
	return []string{f.name, "--resume", s.ID}
}
func (f fakeSource) Delete(session.Session, string) error { return nil }

func testModel(t *testing.T) (model, *db.DB) {
	return testModelWith(t, nil, "/tmp")
}

// testModelWith builds the app with a terminal (nil = none) and the folder sessions live in.
func testModelWith(t *testing.T, term launch.Terminal, dir string) (model, *db.DB) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	claude := fakeSource{name: "claude", list: []session.Session{
		{Agent: "claude", ID: "aaa", Title: "From files", Dir: dir, LastUsed: time.Now()},
	}}
	qoder := fakeSource{name: "qoder", list: []session.Session{
		{Agent: "qoder", ID: "qqq", Title: "Qoder work", Dir: "/tmp", LastUsed: time.Now()},
	}}
	return newModel(store, []session.Source{claude, qoder}, "/home/u", term), store
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	keyQ  = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
)

// run executes cmd and returns every message it produces.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, run(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// deliver feeds our app's own messages from cmd back into m, like Bubble Tea would.
func deliver(m model, cmd tea.Cmd) model {
	for _, msg := range run(cmd) {
		switch msg.(type) {
		case refreshedMsg, openedMsg, agentlist.SelectedMsg:
			next, more := m.Update(msg)
			m = deliver(next.(model), more)
		}
	}
	return m
}

func press(m model, k tea.KeyMsg) (model, tea.Cmd) {
	next, cmd := m.Update(k)
	return next.(model), cmd
}

func quits(cmd tea.Cmd) bool {
	for _, msg := range run(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

func TestFirstRunRefreshesFirstAgent(t *testing.T) {
	m, _ := testModel(t)
	if m.focus != agentPane || m.agent.Name() != "claude" || m.sessions.Len() != 0 {
		t.Fatalf("start: focus %v agent %q sessions %d", m.focus, m.agent.Name(), m.sessions.Len())
	}

	m = deliver(m, m.Init())

	if s, ok := m.sessions.Selected(); !ok || s.Title != "From files" {
		t.Errorf("after refresh selected = %+v", s)
	}
	if !strings.Contains(m.agents.View(), "1") {
		t.Errorf("claude count should be 1 after refresh:\n%s", m.agents.View())
	}
}

func TestMovingInAgentsSwitchesSessions(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())

	m, cmd := press(m, down)
	m = deliver(m, cmd)

	if m.agent.Name() != "qoder" {
		t.Fatalf("agent = %q, want qoder", m.agent.Name())
	}
	if s, ok := m.sessions.Selected(); !ok || s.Title != "Qoder work" {
		t.Errorf("qoder selected = %+v", s)
	}
}

func TestEnterMovesFocusThenChooses(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())

	m, _ = press(m, enter)
	if m.focus != sessionPane {
		t.Fatal("enter on agents should focus the sessions pane")
	}
	m, cmd := press(m, enter)
	if m.chosen == nil || m.chosen.ID != "aaa" || !quits(cmd) {
		t.Errorf("enter on a session should choose it and quit; chosen = %+v", m.chosen)
	}
}

func TestEscGoesBackAndNeverQuits(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())

	m, _ = press(m, enter)
	m, cmd := press(m, esc)
	if m.focus != agentPane || quits(cmd) {
		t.Error("esc in sessions should go back to agents")
	}
	if _, cmd := press(m, esc); quits(cmd) {
		t.Error("esc in agents must not quit")
	}
}

func TestQuitKeys(t *testing.T) {
	m, _ := testModel(t)
	if _, cmd := press(m, keyQ); !quits(cmd) {
		t.Error("q should quit")
	}
	if _, cmd := press(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c should quit")
	}
}

func TestView(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())

	v := m.View()
	for _, want := range []string{"Agents", "Claude Code", "From files", "claude --resume aaa", "refreshed"} {
		if !strings.Contains(v, want) {
			t.Errorf("view is missing %q", want)
		}
	}
}

type fakeTerminal struct {
	opened []string
}

func (f *fakeTerminal) Name() string { return "fake" }
func (f *fakeTerminal) OpenTab(dir, title string, argv []string) error {
	f.opened = append(f.opened, dir+"|"+title+"|"+strings.Join(argv, " "))
	return nil
}

func TestEnterOpensNewTabAndStaysOpen(t *testing.T) {
	term := &fakeTerminal{}
	m, _ := testModelWith(t, term, "/tmp")
	m = deliver(m, m.Init())

	m, _ = press(m, enter)
	m, cmd := press(m, enter)
	if m.chosen != nil {
		t.Fatal("with a terminal, enter must not choose a session to resume here")
	}
	for _, msg := range run(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("with a terminal, enter must keep the TUI open")
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}

	if len(term.opened) != 1 || term.opened[0] != "/tmp|From files|claude --resume aaa" {
		t.Errorf("opened = %q", term.opened)
	}
	if !strings.Contains(m.View(), "opened") {
		t.Errorf("status should say the tab was opened:\n%s", m.View())
	}
}

func TestOResumesHere(t *testing.T) {
	m, _ := testModelWith(t, &fakeTerminal{}, "/tmp")
	m = deliver(m, m.Init())

	m, _ = press(m, enter)
	m, cmd := press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if m.chosen == nil || m.chosen.ID != "aaa" || !quits(cmd) {
		t.Errorf("o should resume here; chosen = %+v", m.chosen)
	}
}

func TestNewTabRefusesMissingFolder(t *testing.T) {
	term := &fakeTerminal{}
	m, _ := testModelWith(t, term, "/nonexistent/folder")
	m = deliver(m, m.Init())

	m, _ = press(m, enter)
	m, cmd := press(m, enter)
	m = deliver(m, cmd)

	if len(term.opened) != 0 {
		t.Errorf("must not open a tab for a missing folder: %q", term.opened)
	}
	if !strings.Contains(m.View(), "folder no longer exists") {
		t.Errorf("status should explain why:\n%s", m.View())
	}
}
