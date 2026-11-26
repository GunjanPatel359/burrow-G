package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/handoff"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/components/agentlist"
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
		path, err := fileFor(dir, s, ".jsonl")
		if err == nil {
			_, err = exportSession(src, s, path, version)
		}
		return exportedMsg{agent: s.Agent, title: s.Title, path: path, err: err}
	}
}

// fileFor is <dir>/<agent>-<id><ext>. Ids come from agents' own files, so
// one that isn't a plain name is refused rather than allowed to lead out of
// dir.
func fileFor(dir string, s session.Session, ext string) (string, error) {
	name := s.Agent + "-" + s.ID
	if s.ID == "" || s.ID == "." || s.ID == ".." || strings.ContainsAny(name, "/\\\x00") {
		return "", fmt.Errorf("session id %q can't be a file name", s.ID)
	}
	return filepath.Join(dir, name+ext), nil
}

// exportSession writes the export of s to path.
func exportSession(src session.Source, s session.Session, path, version string) (portable.End, error) {
	var end portable.End
	err := writePrivate(path, func(f io.Writer) error {
		w := portable.NewWriter(f, version)
		if err := src.Export(s, w); err != nil {
			return err
		}
		var err error
		end, err = w.Close()
		return err
	})
	return end, err
}

// writePrivate writes path, readable only by its owner, in its folder
// (mode 700). It writes a hidden temp file beside it and renames it into
// place only once write succeeded, so path is always whole: a failure,
// a quit midway or two writes at once never leave half a file, and a
// failed rewrite keeps the old one.
func writePrivate(path string, write func(io.Writer) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*") // mode 600
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op once renamed
	err = write(tmp)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// findAgents looks up which agents can be started: the first word of each
// one's StartCommand on PATH.
func findAgents(sources []session.Source, lookPath func(string) (string, error)) tea.Cmd {
	return func() tea.Msg {
		var names []string
		for _, src := range sources {
			if _, err := lookPath(src.StartCommand("")[0]); err == nil {
				names = append(names, src.Name())
			}
		}
		return agentsFoundMsg{names: names}
	}
}

// prepareHandoff prepares continuing s in target: the folder is checked first, so
// nothing is written for a session that can't be continued; then the
// export and the handoff file (spec 0004, section 2).
func prepareHandoff(src, target session.Source, s session.Session, exports, handoffs, version string, here bool) tea.Cmd {
	return func() tea.Msg {
		fail := func(err error) tea.Msg { return handedOffMsg{s: s, target: target.Name(), err: err} }
		if err := session.CheckDir(s); err != nil {
			return fail(err)
		}
		export := filepath.Join(exports, s.Agent+"-"+s.ID+".jsonl")
		if _, err := exportSession(src, s, exports, export, version); err != nil {
			return fail(err)
		}
		path := filepath.Join(handoffs, s.Agent+"-"+s.ID+".md")
		agent := agentlist.DisplayName(s.Agent)
		if err := writeHandoff(export, handoffs, path, agent); err != nil {
			return fail(err)
		}
		argv := target.StartCommand(handoff.Prompt(path, agent, s.Title))
		return handedOffMsg{s: s, target: target.Name(), argv: argv, here: here}
	}
}

// writeHandoff renders the export at from into a private file at path,
// removed again if anything fails.
func writeHandoff(from, dir, path, agent string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	r, err := portable.NewReader(in)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	out, err := createPrivate(path)
	if err != nil {
		return err
	}
	err = handoff.Render(r, handoff.Options{Agent: agent, Record: from}, out)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

// continueInTab starts the target agent in a new terminal tab.
func continueInTab(term launch.Terminal, s session.Session, target string, argv []string) tea.Cmd {
	return func() tea.Msg {
		return continuedMsg{title: s.Title, target: target, err: term.OpenTab(s.Dir, s.Title, argv)}
	}
}
