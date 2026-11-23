// Package event holds the events a command returns to the TUI.
package event

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

// Quit asks the TUI to exit.
type Quit struct{}

// All lists every event type, for tests.
var All = []Event{ThemeChanged{}, Refresh{}, ShowHelp{}, Quit{}}
