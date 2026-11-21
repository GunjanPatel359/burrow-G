package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Background jobs. Only these touch the database or the agents' files.

// openTab opens a session in a new terminal tab.
func openTab(term launch.Terminal, s session.Session, argv []string) tea.Cmd {
	return func() tea.Msg {
		if err := session.CheckDir(s); err != nil {
			return openedMsg{title: s.Title, err: err}
		}
		return openedMsg{title: s.Title, err: term.OpenTab(s.Dir, s.Title, argv)}
	}
}

// refresh re-reads one agent's sessions into the database.
func refresh(store *db.DB, src session.Source) tea.Cmd {
	return func() tea.Msg {
		return refreshedMsg{agent: src.Name(), err: store.Refresh(src)}
	}
}

// bookmark turns a session's bookmark on or off.
func bookmark(store *db.DB, s session.Session, on bool) tea.Cmd {
	return func() tea.Msg {
		return bookmarkedMsg{s: s, on: on, err: store.SetBookmark(s, on)}
	}
}
