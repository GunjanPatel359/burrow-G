package main

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// version is set at build time: go build -ldflags "-X main.version=0.0.1".
var version = "dev"

func versionString() string { return "burrow-G " + version }

const usage = `burrow-G — find and resume your AI coding agent sessions.

Usage:
  burrow-g                 open the terminal UI
  burrow-g list            print every session (refreshes all agents)
  burrow-g resume <id>     resume a session by id or id prefix
  burrow-g version         print the version
  burrow-g help            show this help

In the UI:
  ↑/↓      move              enter    open session (new tab, or here)
  →/tab    go to sessions    o        resume here
  ←/esc    back to agents    /        search
  r        refresh           q        quit

Data: ~/.local/share/burrow-g (agents' own files are only read, never changed)
`

func printUsage(w io.Writer) { fmt.Fprint(w, usage) }

// refreshAll refreshes every agent into the database, then returns all rows.
func refreshAll(store *db.DB, sources []session.Source, warn io.Writer) []session.Session {
	var all []session.Session
	for _, src := range sources {
		if err := store.Refresh(src); err != nil {
			fmt.Fprintf(warn, "warning: %s: %v\n", src.Name(), err)
		}
		list, err := store.List(src.Name())
		if err != nil {
			fmt.Fprintf(warn, "warning: %s: %v\n", src.Name(), err)
			continue
		}
		all = append(all, list...)
	}
	return all
}

// pickSession finds exactly one session by id prefix, plus its agent's source.
func pickSession(store *db.DB, sources []session.Source, prefix string) (session.Session, session.Source, error) {
	matches, err := store.Find(prefix)
	if err != nil {
		return session.Session{}, nil, err
	}
	if len(matches) == 0 {
		return session.Session{}, nil, fmt.Errorf("no session starts with %q (run 'burrow-g list' to refresh)", prefix)
	}
	if len(matches) > 1 {
		return session.Session{}, nil, fmt.Errorf("%d sessions match %q, type more of the id", len(matches), prefix)
	}

	src, err := findSource(sources, matches[0].Agent)
	return matches[0], src, err
}

func findSource(sources []session.Source, agent string) (session.Source, error) {
	for _, src := range sources {
		if src.Name() == agent {
			return src, nil
		}
	}
	return nil, fmt.Errorf("unknown agent %q", agent)
}

func printList(w io.Writer, list []session.Session, home string, now time.Time) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, s := range list {
		title := helpers.Truncate(s.Title, 50)
		if s.Bookmarked {
			title = "★ " + title
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			title,
			s.Agent,
			helpers.ShortPath(s.Dir, home),
			helpers.TimeAgo(s.LastUsed, now),
			shortID(s.ID),
		)
	}
	tw.Flush()
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
