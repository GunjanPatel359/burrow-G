package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/agents/claude"
	"github.com/GunjanPatel359/burrow-G/internal/agents/codex"
	"github.com/GunjanPatel359/burrow-G/internal/agents/copilot"
	"github.com/GunjanPatel359/burrow-G/internal/agents/opencode"
	"github.com/GunjanPatel359/burrow-G/internal/agents/qoder"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/launch"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return
	case "version", "-v", "--version":
		fmt.Println(versionString())
		return
	case "", "list":
	case "resume":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: burrow-g resume <id>")
			os.Exit(2)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		printUsage(os.Stderr)
		os.Exit(2)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fail(err)
	}

	sources := []session.Source{
		claude.Source{Home: filepath.Join(home, ".claude")},
		qoder.Source{Home: filepath.Join(home, ".qoder")},
		opencode.Source{DBPath: filepath.Join(home, ".local", "share", "opencode", "opencode.db")},
		copilot.Source{DBPath: filepath.Join(home, ".copilot", "session-store.db")},
		codex.Source{Home: filepath.Join(home, ".codex")},
	}

	store, err := db.Open(filepath.Join(helpers.DataDir(home), "sessions.db"))
	if err != nil {
		fail(err)
	}
	defer store.Close()

	switch cmd {
	case "list":
		list := refreshAll(store, sources, os.Stderr)
		session.Sort(list, func(s session.Session) bool { return s.Bookmarked })
		printList(os.Stdout, list, home, time.Now())

	case "resume":
		s, src, err := pickSession(store, sources, os.Args[2])
		if err != nil {
			fail(err)
		}
		store.Close()
		fail(session.Resume(s, src.ResumeCommand(s)))

	default:
		chosen, err := tui.Run(store, sources, home, launch.Detect(os.Getenv))
		if err != nil {
			fail(err)
		}
		if chosen == nil {
			return
		}
		src, err := findSource(sources, chosen.Agent)
		if err != nil {
			fail(err)
		}
		store.Close()
		fail(session.Resume(*chosen, src.ResumeCommand(*chosen)))
	}
}

// fail prints err and exits. session.Resume only returns on error, so it ends there too.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "burrow-g:", err)
	os.Exit(1)
}
