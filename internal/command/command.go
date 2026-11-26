// Package command is the command bar's engine: it reads a line, finds the
// command, checks its arguments and runs it. The commands themselves live in
// command/commands, one file each.
package command

import (
	"slices"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Command is one thing the command bar can run.
type Command struct {
	Name string // what you type, e.g. "theme"
	Help string // shown next to it in the list
	Key  string // direct key shown in the list, "" if none
	Args []Arg  // what it takes; at most one without a Name
	Run  func(env Env, in Input) (event.Event, error)
}

// Arg is one argument: written "@name:value", or the bare value when Name is "".
type Arg struct {
	Name     string
	Help     string
	Required bool
	Suggest  func(env Env, typed string) []Suggestion // nil: no suggestions
	Check    func(env Env, v string) error            // nil: any value is fine
}

// Input is what Run gets: the checked values, by argument name.
type Input struct{ values map[string]string }

// Value is the bare value, e.g. "nord" in "theme nord".
func (in Input) Value() string { return in.values[""] }

// Get is the value of @name, "" if it wasn't given.
func (in Input) Get(name string) string { return in.values[name] }

// Suggestion is one row in the list.
type Suggestion struct {
	Text, Hint, Key string
	Preview         event.Event // applied while the row is highlighted
}

// Match keeps rows starting with typed first, then rows containing it.
func Match(rows []Suggestion, typed string) []Suggestion {
	typed = strings.ToLower(typed)
	var prefix, rest []Suggestion
	for _, r := range rows {
		t := strings.ToLower(r.Text)
		switch {
		case strings.HasPrefix(t, typed):
			prefix = append(prefix, r)
		case strings.Contains(t, typed):
			rest = append(rest, r)
		}
	}
	return append(prefix, rest...)
}

// Env is what commands may use from the app.
type Env struct {
	Store   *db.DB
	Themes  []string
	Current func() string // saved theme name
	Sources []session.Source
	Home    string
	Keys    []string // key help lines, for help
	// Selected is the highlighted session, false when there is none.
	Selected func() (session.Session, bool)
	// Targets are the agents a session can be handed to, and whether they
	// have been looked up yet.
	Targets func() ([]string, bool)
}

// Service holds the commands and runs lines against them.
type Service struct {
	env      Env
	commands []Command
}

func NewService(env Env, commands ...Command) *Service {
	return &Service{env: env, commands: commands}
}

// Commands returns every command, in the order they were given.
func (s *Service) Commands() []Command { return slices.Clone(s.commands) }

func (s *Service) find(name string) (Command, bool) {
	for _, c := range s.commands {
		if c.Name == name {
			return c, true
		}
	}
	return Command{}, false
}

// Usage is how c is written, e.g. "theme <value>" or "new @agent: @location:".
func Usage(c Command) string {
	var b strings.Builder
	b.WriteString(c.Name)
	for _, a := range c.Args {
		if a.Name == "" {
			b.WriteString(" <value>")
		} else {
			b.WriteString(" @" + a.Name + ":")
		}
	}
	return b.String()
}

// arg is c's argument with this name ("" for the bare value), nil if none.
func (c Command) arg(name string) *Arg {
	for i := range c.Args {
		if c.Args[i].Name == name {
			return &c.Args[i]
		}
	}
	return nil
}
