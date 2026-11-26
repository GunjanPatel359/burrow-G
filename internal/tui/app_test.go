package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentpicker"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/commandbar"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
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
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
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
	return newModel(store, []session.Source{claude, qoder}, "/home/u", term, theme.Env{}, alwaysDark), store
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
		case refreshedMsg, openedMsg, bookmarkedMsg, agentlist.SelectedMsg,
			commandbar.RanMsg, commandbar.PreviewMsg, commandbar.ClosedMsg,
			exportedMsg, handoffReadyMsg, agentpicker.PickedMsg, agentpicker.ClosedMsg:
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
	if m.chosen == nil || m.chosen.Session.ID != "aaa" || !quits(cmd) {
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
	if m.chosen == nil || m.chosen.Session.ID != "aaa" || !quits(cmd) {
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

var bKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}

// twoSessions shows "Newer" above "Older" with the sessions pane focused.
func twoSessions(t *testing.T) (model, *db.DB) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	claude := fakeSource{name: "claude", list: []session.Session{
		{Agent: "claude", ID: "new", Title: "Newer", Dir: "/tmp", LastUsed: time.Now()},
		{Agent: "claude", ID: "old", Title: "Older", Dir: "/tmp", LastUsed: time.Now().Add(-time.Hour)},
	}}
	m := newModel(store, []session.Source{claude}, "/home/u", nil, theme.Env{}, alwaysDark)
	m = deliver(m, m.Init())
	m, _ = press(m, enter)
	return m, store
}

func firstRow(m model) string { return strings.Split(m.sessions.View(), "\n")[1] }

func TestBookmarkMovesToTopAndBack(t *testing.T) {
	m, _ := twoSessions(t)
	m, _ = press(m, down)

	m, cmd := press(m, bKey)
	m = deliver(m, cmd)
	if s, _ := m.sessions.Selected(); s.ID != "old" || !s.Bookmarked {
		t.Errorf("cursor should stay on Older, now bookmarked; got %+v", s)
	}
	if row := firstRow(m); !strings.Contains(row, "★") || !strings.Contains(row, "Older") {
		t.Errorf("Older should be on top with ★: %q", row)
	}
	if !strings.Contains(m.View(), "bookmarked") {
		t.Errorf("status should say it was bookmarked:\n%s", m.View())
	}

	m, cmd = press(m, bKey)
	m = deliver(m, cmd)
	if s, _ := m.sessions.Selected(); s.ID != "old" || s.Bookmarked {
		t.Errorf("cursor should stay on Older, no longer bookmarked; got %+v", s)
	}
	if row := firstRow(m); strings.Contains(row, "★") || !strings.Contains(row, "Newer") {
		t.Errorf("Newer should be back on top: %q", row)
	}
}

func TestBookmarkSurvivesRefreshAndRestart(t *testing.T) {
	m, store := twoSessions(t)
	m, _ = press(m, down)
	m, cmd := press(m, bKey)
	m = deliver(m, cmd)

	m, cmd = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = deliver(m, cmd)
	if row := firstRow(m); !strings.Contains(row, "★") || !strings.Contains(row, "Older") {
		t.Errorf("after refresh: %q", row)
	}

	again := newModel(store, m.sources, "/home/u", nil, theme.Env{}, alwaysDark)
	if row := firstRow(again); !strings.Contains(row, "★") || !strings.Contains(row, "Older") {
		t.Errorf("after restart: %q", row)
	}
}

func TestBookmarkIgnoredWhileSearchingOrInAgents(t *testing.T) {
	m, store := twoSessions(t)
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m, cmd := press(m, bKey)
	m = deliver(m, cmd)
	if !m.sessions.Searching() {
		t.Error("b while searching should be typed into the search box")
	}

	m, _ = twoSessions(t)
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyLeft})
	m, cmd = press(m, bKey)
	m = deliver(m, cmd)

	list, _ := store.List("claude")
	for _, s := range list {
		if s.Bookmarked {
			t.Errorf("%s got bookmarked", s.ID)
		}
	}
}

var ctrlK = tea.KeyMsg{Type: tea.KeyCtrlK}

// typeText types s one key at a time, dropping the input's own commands.
func typeText(m model, s string) model {
	for _, r := range s {
		m, _ = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

// pressAll presses k and feeds back what the bar and app send.
func pressAll(m model, msg tea.Msg) model {
	next, cmd := m.Update(msg)
	return deliver(next.(model), cmd)
}

func lines(v string) int { return strings.Count(v, "\n") + 1 }

func TestCtrlKOpensAndRunsTheme(t *testing.T) {
	m, store := testModel(t)
	m = pressAll(m, ctrlK)
	if !m.bar.Opened() {
		t.Fatal("ctrl+k should open the bar")
	}
	m = typeText(m, "theme nord")
	m = pressAll(m, enter)

	if m.bar.Opened() || m.theme.Name != "nord" {
		t.Errorf("bar open %v, theme %q", m.bar.Opened(), m.theme.Name)
	}
	if got, _ := store.Setting("theme"); got != "nord" {
		t.Errorf("saved setting = %q", got)
	}
	if !strings.Contains(m.status.View(), `theme "nord"`) {
		t.Errorf("status = %q", m.status.View())
	}
}

func TestPreviewRevertsOnClose(t *testing.T) {
	m, _ := testModel(t)
	m = typeText(pressAll(m, ctrlK), "theme ")
	m = pressAll(m, down)
	if m.theme.Name == theme.Default {
		t.Fatal("down should preview another theme")
	}
	m = pressAll(m, esc)
	m = pressAll(m, esc)
	if m.bar.Opened() || m.theme.Name != theme.Default {
		t.Errorf("bar open %v, theme %q", m.bar.Opened(), m.theme.Name)
	}
}

func TestEnterAppliesPickedTheme(t *testing.T) {
	m, store := testModel(t)
	m = typeText(pressAll(m, ctrlK), "theme ")
	m = pressAll(m, down)
	picked := m.theme.Name
	m = pressAll(m, enter)

	if m.bar.Opened() || m.theme.Name != picked {
		t.Errorf("bar open %v, theme %q, want %q", m.bar.Opened(), m.theme.Name, picked)
	}
	if got, _ := store.Setting("theme"); got != picked {
		t.Errorf("saved setting = %q, want %q", got, picked)
	}
}

func TestRunErrorKeepsBarOpen(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(typeText(pressAll(m, ctrlK), "theme zz"), enter)

	if !m.bar.Opened() || !strings.Contains(m.View(), `no theme "zz"`) {
		t.Errorf("bar open %v; want the error shown", m.bar.Opened())
	}
	if n := lines(m.View()); n != m.height {
		t.Errorf("view has %d lines with an error, want %d", n, m.height)
	}
	m = pressAll(m, ctrlK)
	if m.bar.Opened() || m.theme.Name != theme.Default {
		t.Errorf("ctrl+k: bar open %v, theme %q", m.bar.Opened(), m.theme.Name)
	}
}

func TestSavedThemeLoadedAtStart(t *testing.T) {
	m, store := testModel(t)
	if err := store.SetSetting("theme", "dracula"); err != nil {
		t.Fatal(err)
	}
	m = newModel(store, m.sources, "/home/u", nil, theme.Env{}, alwaysDark)
	if m.theme.Name != "dracula" {
		t.Errorf("theme = %q", m.theme.Name)
	}
}

func TestUnknownSavedThemeFallsBack(t *testing.T) {
	m, store := testModel(t)
	if err := store.SetSetting("theme", "gone"); err != nil {
		t.Fatal(err)
	}
	m = newModel(store, m.sources, "/home/u", nil, theme.Env{}, alwaysDark)
	if m.theme.Name != theme.Default || strings.Contains(m.status.View(), "error") {
		t.Errorf("theme %q, status %q", m.theme.Name, m.status.View())
	}
}

func TestEveryEventHandled(t *testing.T) {
	for _, e := range event.All {
		m, _ := testModel(t)
		m.apply(e)
		if strings.Contains(m.status.View(), "unknown event") {
			t.Errorf("%T is not handled", e)
		}
	}
	m, _ := testModel(t)
	m.apply(struct{}{})
	if !strings.Contains(m.status.View(), "unknown event") {
		t.Errorf("unknown events should be reported: %q", m.status.View())
	}
}

func TestHelpTakesOverRightSide(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())
	m = typeText(pressAll(m, ctrlK), "help")
	m = pressAll(m, enter)

	v := m.View()
	if !strings.Contains(v, "Commands") || !strings.Contains(v, "show keys and commands") || strings.Contains(v, "From files") {
		t.Errorf("help should replace the right side:\n%s", v)
	}
	if _, cmd := press(m, keyQ); quits(cmd) {
		t.Error("q must be ignored while help is shown")
	}
	m = pressAll(m, esc)
	if v := m.View(); !strings.Contains(v, "From files") || strings.Contains(v, "Commands") {
		t.Errorf("esc should bring the sessions back:\n%s", v)
	}
}

func TestHelpScrollsWithinBounds(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m.apply(event.ShowHelp{Text: strings.Repeat("line\n", 20)})
	up := tea.KeyMsg{Type: tea.KeyUp}

	m = pressAll(m, up)
	if m.helpTop != 0 {
		t.Errorf("up at the top: helpTop = %d", m.helpTop)
	}
	for range 50 {
		m = pressAll(m, down)
	}
	if want := len(m.help) - m.bodyHeight(); m.helpTop != want {
		t.Errorf("helpTop = %d, want %d", m.helpTop, want)
	}
}

func TestBarFitsShortTerminal(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m = typeText(pressAll(m, ctrlK), "")
	if len(m.bar.Rows()) == 0 {
		t.Fatal("empty line should suggest every command")
	}
	v := m.View()
	if n := strings.Count(v, "\n"); n != 9 {
		t.Errorf("view has %d newlines, want 9:\n%s", n, v)
	}
	if !strings.Contains(v, "theme") || !strings.Contains(v, "> ") {
		t.Errorf("rows and input should show:\n%s", v)
	}
}

func TestKeysIgnoredWhileBarOpen(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, ctrlK)
	m, cmd := press(m, keyQ)
	if quits(cmd) {
		t.Error("q must not quit while the bar is open")
	}
	if !strings.Contains(m.bar.View(), "q") {
		t.Errorf("input should show q: %q", m.bar.View())
	}
}

func TestPreviewRevertsWhenOtherCommandRuns(t *testing.T) {
	m, store := testModel(t)
	m = typeText(pressAll(m, ctrlK), "theme ")
	m = pressAll(m, down)
	for range len("theme ") {
		m, _ = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m = typeText(m, "refresh")
	m = pressAll(m, enter)

	if m.bar.Opened() || m.theme.Name != theme.Default {
		t.Errorf("bar open %v, theme %q, want %q", m.bar.Opened(), m.theme.Name, theme.Default)
	}
	if got, _ := store.Setting("theme"); got != "" {
		t.Errorf("setting = %q, want unset", got)
	}
}

func TestCtrlCQuitsWithBarOpen(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, ctrlK)
	if _, cmd := press(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c should quit while the bar is open")
	}
}

func TestCtrlCQuitsInHelp(t *testing.T) {
	m, _ := testModel(t)
	m.apply(event.ShowHelp{Text: "line\n"})
	if _, cmd := press(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c should quit while help is shown")
	}
}

func TestHelpTopClampedOnResize(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 10})
	m.apply(event.ShowHelp{Text: strings.Repeat("line\n", 20)})
	for range 50 {
		m = pressAll(m, down)
	}
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 20})
	if want := len(m.help) - m.bodyHeight(); m.helpTop != want {
		t.Errorf("helpTop = %d, want %d", m.helpTop, want)
	}
}

func TestHelpLinesTruncatedNotWrapped(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, tea.WindowSizeMsg{Width: 60, Height: 10})
	text := ""
	for i := range 19 {
		text += fmt.Sprintf("line%d\n", i)
	}
	text += strings.Repeat("x", 200) + "\nlast line\n"
	m.apply(event.ShowHelp{Text: text})
	for range 50 {
		m = pressAll(m, down)
	}
	if v := m.View(); !strings.Contains(v, "last line") || lines(v) != m.height {
		t.Errorf("the last help line should be reachable:\n%s", v)
	}
}

func TestThemePaintsBackground(t *testing.T) {
	m, _ := testModel(t)
	var painted []lipgloss.Color
	m.paint = func(c lipgloss.Color) { painted = append(painted, c) }
	last := func() lipgloss.Color { return painted[len(painted)-1] }

	m = pressAll(typeText(pressAll(m, ctrlK), "theme "), down) // preview
	if len(painted) == 0 || last() == theme.Get(theme.Default, theme.Env{}).Background {
		t.Fatalf("preview should paint the previewed background, painted %v", painted)
	}
	m = pressAll(pressAll(m, esc), esc) // close → back to the saved theme
	if last() != theme.Get(theme.Default, theme.Env{}).Background {
		t.Errorf("closing should paint the saved background, got %q", last())
	}
	m = pressAll(typeText(pressAll(m, ctrlK), "theme nord"), enter)
	if last() != theme.Get("nord", theme.Env{}).Background {
		t.Errorf("theme nord should paint %q, got %q", theme.Get("nord", theme.Env{}).Background, last())
	}
}

func alwaysDark() bool { return true }

// With no saved theme a light terminal starts light; a saved one always wins.
func TestFirstThemeFollowsTerminal(t *testing.T) {
	m, store := testModel(t)
	light := func() bool { return false }
	if got := newModel(store, m.sources, "/home/u", nil, theme.Env{}, light).theme.Name; got != theme.DefaultLight {
		t.Errorf("light terminal started on %q, want %q", got, theme.DefaultLight)
	}
	if err := store.SetSetting("theme", "nord"); err != nil {
		t.Fatal(err)
	}
	if got := newModel(store, m.sources, "/home/u", nil, theme.Env{}, light).theme.Name; got != "nord" {
		t.Errorf("saved nord started on %q", got)
	}
}

// At 16 colors the terminal keeps its own background, also while previewing.
func TestColor16NeverPaints(t *testing.T) {
	_, store := testModel(t)
	m := newModel(store, nil, "/home/u", nil, theme.Env{Depth: theme.Color16}, alwaysDark)
	var painted []lipgloss.Color
	m.paint = func(c lipgloss.Color) { painted = append(painted, c) }
	m = pressAll(typeText(pressAll(m, ctrlK), "theme nord"), enter)
	for _, c := range painted {
		if c != "" {
			t.Errorf("painted %q at 16 colors", c)
		}
	}
}

// Hints and the help command's key list come from the same bindings, with
// arrows drawn from the glyph set.
func TestHintsComeFromBindings(t *testing.T) {
	_, store := testModel(t)
	sources := []session.Source{fakeSource{name: "claude"}}
	m := newModel(store, sources, "/home/u", nil, theme.Env{ASCII: true}, alwaysDark)

	if v := m.status.View(); !strings.Contains(v, "^/v agent") || !strings.Contains(v, "enter/> sessions") {
		t.Errorf("agent hints = %q", v)
	}
	m = pressAll(m, tea.KeyMsg{Type: tea.KeyRight})
	if v := m.status.View(); !strings.Contains(v, "^/v move") || !strings.Contains(v, "< agents") {
		t.Errorf("session hints = %q", v)
	}
	m = pressAll(typeText(pressAll(m, ctrlK), "help"), enter)
	help := strings.Join(m.help, "\n")
	for _, want := range []string{"^/v agent", "^/v move", "enter resume", "ctrl+k commands"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q:\n%s", want, help)
		}
	}
	if strings.Count(help, "q quit") != 1 {
		t.Errorf("help should list each key once:\n%s", help)
	}
}

func TestNarrowWindowShowsOnePane(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())
	m = pressAll(m, tea.WindowSizeMsg{Width: 50, Height: 20})

	v := m.View()
	if !strings.Contains(v, "Agents") || strings.Contains(v, "From files") || strings.Contains(v, "│") {
		t.Errorf("narrow window should show only the agents pane:\n%s", v)
	}
	m = pressAll(m, tea.KeyMsg{Type: tea.KeyRight})
	v = m.View()
	if strings.Contains(v, "Agents") || !strings.Contains(v, "From files") {
		t.Errorf("→ should switch to the sessions pane:\n%s", v)
	}
	if !strings.Contains(m.status.View(), "← agents") {
		t.Errorf("session hints should offer the way back: %q", m.status.View())
	}
	if lines(v) != m.height {
		t.Errorf("view is %d lines, want %d", lines(v), m.height)
	}
}

func TestShortWindowHidesDetails(t *testing.T) {
	m, _ := testModel(t)
	m = deliver(m, m.Init())
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 20})
	if !strings.Contains(m.View(), "resume") {
		t.Fatalf("details should show in a tall window:\n%s", m.View())
	}
	m = pressAll(m, tea.WindowSizeMsg{Width: 80, Height: 11})
	v := m.View()
	if strings.Contains(v, "folder") || !strings.Contains(v, "From files") {
		t.Errorf("short window should hide details but keep the list:\n%s", v)
	}
	if lines(v) != m.height {
		t.Errorf("view is %d lines, want %d", lines(v), m.height)
	}
}

func TestTooSmallWindow(t *testing.T) {
	m, _ := testModel(t)
	m = pressAll(m, tea.WindowSizeMsg{Width: 39, Height: 20})
	v := m.View()
	if !strings.Contains(v, "make the window bigger") || !strings.Contains(v, "needs 40×8") || strings.Contains(v, "Agents") {
		t.Errorf("tiny window should show only the note:\n%s", v)
	}
	if _, cmd := press(m, keyQ); !quits(cmd) {
		t.Error("q should still quit")
	}
	m = pressAll(m, tea.WindowSizeMsg{Width: 40, Height: 8})
	if strings.Contains(m.View(), "make the window bigger") {
		t.Errorf("40×8 is big enough:\n%s", m.View())
	}
}

func TestEmptyStates(t *testing.T) {
	_, store := testModel(t)
	m := newModel(store, []session.Source{fakeSource{name: "codex"}}, "/home/u", nil, theme.Env{}, alwaysDark)
	m = deliver(m, m.Init())
	if v := m.View(); !strings.Contains(v, "no sessions yet") || !strings.Contains(v, "start one in Codex and press r") {
		t.Errorf("an agent with no sessions should say how to get one:\n%s", v)
	}

	m = newModel(store, nil, "/home/u", nil, theme.Env{}, alwaysDark)
	if v := m.View(); !strings.Contains(v, "no agents found") || !strings.Contains(v, "burrow-g reads ~/.claude") {
		t.Errorf("no agents should say where burrow-g looks:\n%s", v)
	}
}

func TestBusyStatusSpins(t *testing.T) {
	m, _ := testModel(t)
	var tick tea.Msg
	for _, msg := range run(m.Init()) {
		if _, ok := msg.(spinner.TickMsg); ok {
			tick = msg
		}
	}
	if tick == nil {
		t.Fatal("Init should start the spinner of the first refresh")
	}
	before := m.status.View()
	next, cmd := m.Update(tick)
	if cmd == nil || next.(model).status.View() == before {
		t.Errorf("a tick should move the spinner: %q", before)
	}
}

func TestThemeWithoutColorsSaysWhy(t *testing.T) {
	for _, d := range []theme.Depth{theme.Color16, theme.NoColor} {
		_, store := testModel(t)
		m := newModel(store, nil, "/home/u", nil, theme.Env{Depth: d}, alwaysDark)
		m = pressAll(typeText(pressAll(m, ctrlK), "theme nord"), enter)
		if v := m.status.View(); !strings.Contains(v, `theme "nord" (colors need truecolor or 256)`) {
			t.Errorf("depth %v: status = %q", d, v)
		}
		if got, _ := store.Setting("theme"); got != "nord" {
			t.Errorf("depth %v: the choice should still be saved, got %q", d, got)
		}
	}
}
