package tui

import "github.com/GunjanPatel359/burrow-G/internal/session"

// Messages the app sends to itself. Components keep their own (e.g. agentlist.SelectedMsg).

// refreshedMsg arrives when an agent's sessions were re-read into the database.
type refreshedMsg struct {
	agent string
	err   error
}

// openedMsg arrives when a session was opened in a new terminal tab.
type openedMsg struct {
	title string
	err   error
}

// bookmarkedMsg arrives when a session's bookmark was turned on or off.
type bookmarkedMsg struct {
	s   session.Session
	on  bool
	err error
}

// exportedMsg arrives when a session was exported to path.
type exportedMsg struct {
	agent string
	title string
	path  string
	err   error
}

// agentsFoundMsg arrives with the agents whose CLI is on PATH.
type agentsFoundMsg struct {
	names []string
}

// handedOffMsg arrives when a handoff file is ready and argv would start the
// target agent with it; err when it couldn't be prepared.
type handedOffMsg struct {
	s      session.Session
	target string
	argv   []string
	here   bool
	err    error
}

// continuedMsg arrives when the target agent was started in a new tab.
type continuedMsg struct {
	title  string
	target string
	err    error
}
