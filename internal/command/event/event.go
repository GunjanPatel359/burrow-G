// Package event holds the events a command returns to the TUI.
package event

import "github.com/GunjanPatel359/burrow-G/internal/session"

// Event is any value a command returns.
type Event any

// ThemeChanged asks the TUI to apply a theme, as a preview or for good.
type ThemeChanged struct {
	Name    string
	Preview bool
}

// Refresh asks the TUI to reload its data.
type Refresh struct{}

// ShowHelp asks the TUI to show help text.
type ShowHelp struct{ Text string }

// ExportRequested asks the TUI to export a session to Markdown.
type ExportRequested struct{ Session session.Session }

// HandoffRequested asks the TUI to hand a session to another agent.
// An empty Agent means: let the person pick one.
type HandoffRequested struct {
	Session session.Session
	Agent   string
}

// Quit asks the TUI to exit.
type Quit struct{}

// All lists every event type, for tests.
var All = []Event{ThemeChanged{}, Refresh{}, ShowHelp{}, ExportRequested{}, HandoffRequested{}, Quit{}}
