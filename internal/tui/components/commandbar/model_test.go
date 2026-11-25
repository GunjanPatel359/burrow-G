package commandbar

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/commands"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func newBar(t *testing.T) Model {
	t.Helper()
	return newBarWith(t, []string{"dracula", "nord"})
}

func newBarWith(t *testing.T, themes []string) Model {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "sessions.db"), "0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	svc := commands.New(command.Env{
		Store:   store,
		Themes:  themes,
		Current: func() string { return "" },
	})
	m := New(theme.Get(theme.Default, theme.Env{}), svc)
	m.SetWidth(60)
	m.Open()
	return m
}

func press(m Model, key tea.KeyMsg) (Model, tea.Msg) {
	m, cmd := m.Update(key)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

var (
	keyTab   = tea.KeyMsg{Type: tea.KeyTab}
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyCtrlK = tea.KeyMsg{Type: tea.KeyCtrlK}
)

func TestTypingShowsSuggestions(t *testing.T) {
	m := typeText(newBar(t), "th")
	rows := m.Rows()
	if len(rows) != 1 || !strings.Contains(rows[0], "theme") {
		t.Errorf("rows = %q", rows)
	}
}

func TestTabCompletes(t *testing.T) {
	m, _ := press(typeText(newBar(t), "th"), keyTab)
	if got := m.input.Value(); got != "theme " {
		t.Errorf("input = %q", got)
	}
	m, _ = press(typeText(newBar(t), "theme d"), keyTab)
	if got := m.input.Value(); got != "theme dracula " {
		t.Errorf("input = %q", got)
	}
}

func TestDownEmitsPreview(t *testing.T) {
	m := typeText(newBar(t), "theme ")
	_, msg := press(m, keyDown)
	want := PreviewMsg{Event: event.ThemeChanged{Name: "nord", Preview: true}}
	if msg != want {
		t.Errorf("msg = %#v", msg)
	}
}

func TestEnterRuns(t *testing.T) {
	m, msg := press(typeText(newBar(t), "quit"), keyEnter)
	if msg != (RanMsg{Event: event.Quit{}}) || m.Opened() {
		t.Errorf("msg = %#v, opened = %v", msg, m.Opened())
	}
}

func TestEnterErrorKeepsOpen(t *testing.T) {
	m, _ := press(typeText(newBar(t), "nope"), keyEnter)
	if !m.Opened() || !strings.Contains(m.View(), "unknown command") {
		t.Errorf("view = %q", m.View())
	}
}

func TestEscClearsThenCloses(t *testing.T) {
	m, _ := press(typeText(newBar(t), "th"), keyEsc)
	if !m.Opened() || m.input.Value() != "" {
		t.Fatalf("opened = %v, input = %q", m.Opened(), m.input.Value())
	}
	m, msg := press(m, keyEsc)
	if msg != (ClosedMsg{}) || m.Opened() {
		t.Errorf("msg = %#v, opened = %v", msg, m.Opened())
	}
}

func TestCtrlKCloses(t *testing.T) {
	m, msg := press(newBar(t), keyCtrlK)
	if msg != (ClosedMsg{}) || m.Opened() {
		t.Errorf("msg = %#v, opened = %v", msg, m.Opened())
	}
}

func TestTabKeepsOpeningQuote(t *testing.T) {
	m, _ := press(typeText(newBar(t), `theme "d`), keyTab)
	if got := m.input.Value(); got != `theme "dracula" ` {
		t.Errorf("input = %q", got)
	}
}

// previews runs cmd and returns the PreviewMsgs it sends.
func previews(cmd tea.Cmd) []PreviewMsg {
	if cmd == nil {
		return nil
	}
	var out []PreviewMsg
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			out = append(out, previews(c)...)
		}
	case PreviewMsg:
		out = append(out, msg)
	}
	return out
}

func TestTypingPreviewsFirstRow(t *testing.T) {
	m, _ := press(typeText(newBar(t), "theme "), keyDown)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	want := []PreviewMsg{{Event: event.ThemeChanged{Name: "dracula", Preview: true}}}
	if got := previews(cmd); len(got) != 1 || got[0] != want[0] {
		t.Errorf("after typing d: %#v", got)
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got := previews(cmd); len(got) != 1 || got[0] != (PreviewMsg{}) {
		t.Errorf("no rows should send a nil preview: %#v", got)
	}
}

func TestEnterRunsHighlightedCommand(t *testing.T) {
	m, _ := press(newBar(t), keyDown) // theme → refresh
	m, msg := press(m, keyEnter)
	if msg != (RanMsg{Event: event.Refresh{}}) || m.Opened() {
		t.Errorf("msg = %#v, opened = %v", msg, m.Opened())
	}
}

func TestEnterOnCommandNeedingValueShowsChoices(t *testing.T) {
	m, _ := press(newBar(t), keyEnter) // first row is theme
	if !m.Opened() || m.err != nil || m.input.Value() != "theme " {
		t.Fatalf("opened = %v, err = %v, input = %q", m.Opened(), m.err, m.input.Value())
	}
	if rows := m.Rows(); len(rows) != 2 || !strings.Contains(rows[0], "dracula") {
		t.Errorf("rows = %q, want the themes", rows)
	}
}

func TestEnterPicksHighlightedValue(t *testing.T) {
	m, _ := press(typeText(newBar(t), "theme "), keyDown) // dracula → nord
	m, msg := press(m, keyEnter)
	if msg != (RanMsg{Event: event.ThemeChanged{Name: "nord"}}) || m.Opened() {
		t.Errorf("msg = %#v, opened = %v", msg, m.Opened())
	}
}

func TestEnterCompletesPartWord(t *testing.T) {
	_, msg := press(typeText(newBar(t), "theme d"), keyEnter)
	if msg != (RanMsg{Event: event.ThemeChanged{Name: "dracula"}}) {
		t.Errorf("msg = %#v", msg)
	}
}

func TestEnterRunsFullLineAsTyped(t *testing.T) {
	_, msg := press(typeText(newBar(t), "theme nord "), keyEnter)
	if msg != (RanMsg{Event: event.ThemeChanged{Name: "nord"}}) {
		t.Errorf("msg = %#v", msg)
	}
}

func TestTabPreviewsCompletedValue(t *testing.T) {
	// nord isn't row 0 after the refresh, so this tells the two apart.
	for _, c := range []struct{ line, want string }{{"theme d", "dracula"}, {"theme n", "nord"}} {
		_, cmd := typeText(newBar(t), c.line).Update(keyTab)
		want := PreviewMsg{Event: event.ThemeChanged{Name: c.want, Preview: true}}
		if got := previews(cmd); len(got) != 1 || got[0] != want {
			t.Errorf("%q + tab: %#v", c.line, got)
		}
	}
}

func TestListScrollsPastSixRows(t *testing.T) {
	themes := []string{"t01", "t02", "t03", "t04", "t05", "t06", "t07", "t08", "t09", "t10"}
	m := typeText(newBarWith(t, themes), "theme ")
	if rows := m.Rows(); len(rows) != 6 || !strings.Contains(rows[0], "t01") {
		t.Fatalf("rows = %q, want the first six", rows)
	}
	for range 9 {
		m, _ = press(m, keyDown)
	}
	rows := m.Rows()
	if len(rows) != 6 || !strings.Contains(rows[5], "t10") || !strings.Contains(rows[0], "t05") {
		t.Errorf("after 9 downs rows = %q, want t05..t10", rows)
	}
	_, msg := press(m, keyEnter)
	if msg != (RanMsg{Event: event.ThemeChanged{Name: "t10"}}) {
		t.Errorf("enter on the last row: %#v", msg)
	}
}
