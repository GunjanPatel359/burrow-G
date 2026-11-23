// Package sessionlist is the right pane: one agent's sessions, with search.
package sessionlist

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

type item struct {
	s    session.Session
	home string
}

func (i item) FilterValue() string {
	return i.s.Title + " " + helpers.ShortPath(i.s.Dir, i.home)
}

type Model struct {
	list    list.Model
	theme   theme.Theme
	focused bool
	home    string
	w, h    int
}

func New(t theme.Theme, home string) Model {
	l := list.New(nil, delegate{theme: t}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.DisableQuitKeybindings()
	return Model{list: l, theme: t, home: home}
}

func (m *Model) SetSessions(sessions []session.Session) {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = item{s: s, home: m.home}
	}
	// With a search applied, SetItems returns the new search results as a
	// command; run it now so the list never shows up empty.
	if cmd := m.list.SetItems(items); cmd != nil {
		m.list, _ = m.list.Update(cmd())
	}
}

func (m *Model) SetSize(w, h int) {
	m.w, m.h = w, h
	m.fit()
}

// fit sizes the list; the hint line above it takes the row bubbles' filter line uses.
func (m *Model) fit() {
	h := m.h
	if !m.Filtered() {
		h--
	}
	m.list.SetSize(m.w, max(h, 0))
}

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	m.list.SetDelegate(delegate{theme: m.theme, active: m.focused})
}

func (m *Model) SetTheme(t theme.Theme) {
	m.theme = t
	m.list.SetDelegate(delegate{theme: m.theme, active: m.focused})
}

func (m *Model) ResetCursor() { m.list.ResetSelected() }

// Select moves the cursor to the session with this key, if it is shown.
func (m *Model) Select(key string) {
	for i, it := range m.list.VisibleItems() {
		if it.(item).s.Key() == key {
			m.list.Select(i)
			return
		}
	}
}

func (m Model) Selected() (session.Session, bool) {
	it, ok := m.list.SelectedItem().(item)
	return it.s, ok
}

func (m Model) Len() int { return len(m.list.Items()) }

// Searching is true while typing in the search box.
func (m Model) Searching() bool { return m.list.FilterState() == list.Filtering }

// Filtered is true while a search narrows the list (typing or applied).
func (m Model) Filtered() bool { return m.list.FilterState() != list.Unfiltered }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.list.SetShowFilter(m.Filtered())
	m.fit()
	return m, cmd
}

func (m Model) View() string {
	if m.Filtered() {
		return m.list.View()
	}
	return m.theme.Muted.Render("/ search") + "\n" + m.list.View()
}

// delegate draws one row: "★ Title ......... 2 hours ago".
type delegate struct {
	theme  theme.Theme
	active bool
}

func (delegate) Height() int                         { return 1 }
func (delegate) Spacing() int                        { return 0 }
func (delegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d delegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	s := li.(item).s
	star := theme.Part{Text: "  "}
	if s.Bookmarked {
		star = theme.Part{Text: "★ ", Style: d.theme.Star}
	}
	ago := helpers.TimeAgo(s.LastUsed, time.Now())
	title := helpers.Truncate(s.Title, m.Width()-len(ago)-6)

	left := []theme.Part{star, {Text: title}}
	right := []theme.Part{{Text: ago, Style: d.theme.Muted}}
	fmt.Fprint(w, d.theme.Row(left, right, m.Width(), index == m.Index(), d.active))
}
