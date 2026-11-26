package tui

import (
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

// Messages the app sends to itself. Components keep their own (e.g. agentlist.SelectedMsg).

// refreshedMsg arrives when an agent's sessions were re-read into the database.
type refreshedMsg struct {
	agent string
	err   error
}

// openedMsg arrives when a session was opened in a new terminal tab.
// report, when set, replaces the default success line (handoffs).
type openedMsg struct {
	title  string
	report string
	err    error
}

// exportedMsg arrives when a session was written to an export file.
type exportedMsg struct {
	path  string
	stats transcript.Stats
	err   error
}

// handoffReadyMsg arrives when a handoff prompt is built and ready to launch.
type handoffReadyMsg struct {
	s     session.Session
	agent string
	argv  []string
	stats transcript.Stats
	here  bool
	err   error
}

// bookmarkedMsg arrives when a session's bookmark was turned on or off.
type bookmarkedMsg struct {
	s   session.Session
	on  bool
	err error
}
