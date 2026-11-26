package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

// Background jobs. Only these touch the database or the agents' files.

// openTab opens a session in a new terminal tab.
// report is shown on success instead of the default line; "" keeps the default.
func openTab(term launch.Terminal, s session.Session, argv []string, report string) tea.Cmd {
	return func() tea.Msg {
		if err := session.CheckDir(s); err != nil {
			return openedMsg{title: s.Title, err: err}
		}
		return openedMsg{title: s.Title, report: report, err: term.OpenTab(s.Dir, s.Title, argv)}
	}
}

// exportSession reads a session's conversation and saves it as Markdown
// in burrow-G's data folder.
func exportSession(src session.Transcriber, s session.Session, home string) tea.Cmd {
	return func() tea.Msg {
		t, err := src.Transcript(s)
		if err == nil {
			err = transcript.CheckMessages(t)
		}
		if err != nil {
			return exportedMsg{err: err}
		}
		md, st := transcript.Markdown(t, time.Now())
		path, err := transcript.SaveExport(home, s, md)
		return exportedMsg{path: path, stats: st, err: err}
	}
}

// findTargets looks up which agents can take a handoff. It stats PATH, so it
// runs as a job, not inside Update.
func findTargets(sources []session.Source, lookPath func(string) (string, error)) tea.Cmd {
	return func() tea.Msg {
		var names []string
		for _, src := range transcript.Targets(sources, lookPath) {
			names = append(names, src.Name())
		}
		return targetsMsg{names: names}
	}
}

// prepareHandoff builds the argv that starts agent with s's conversation.
func prepareHandoff(src, dst session.Transcriber, s session.Session, agent string, here bool, lookPath func(string) (string, error)) tea.Cmd {
	return func() tea.Msg {
		argv, st, err := transcript.Handoff(src, dst, s, lookPath)
		return handoffReadyMsg{s: s, agent: agent, argv: argv, stats: st, here: here, err: err}
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
