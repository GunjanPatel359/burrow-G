package tui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
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

// export writes a session to <dir>/<agent>-<id>.jsonl in the portable format.
func export(src session.Source, s session.Session, dir, version string) tea.Cmd {
	return func() tea.Msg {
		path := filepath.Join(dir, s.Agent+"-"+s.ID+".jsonl")
		end, err := exportSession(src, s, dir, path, version)
		return exportedMsg{agent: s.Agent, path: path, records: end.Records, err: err}
	}
}

// exportSession writes the export to a private file. A failed export
// removes the file, so a file in the folder is always a whole session.
func exportSession(src session.Source, s session.Session, dir, path, version string) (portable.End, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return portable.End{}, err
	}
	f, err := createPrivate(path)
	if err != nil {
		return portable.End{}, err
	}
	w := portable.NewWriter(f, version)
	err = src.Export(s, w)
	var end portable.End
	if err == nil {
		end, err = w.Close()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
	}
	return end, err
}

// createPrivate opens path for writing as mode 600. An existing file is
// narrowed to 600 before it is emptied, so old content is never readable
// by others even for a moment.
func createPrivate(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Truncate(0); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
