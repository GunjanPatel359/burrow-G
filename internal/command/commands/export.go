package commands

import (
	"errors"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var errNoSelection = errors.New("select a session first")

// Export saves the selected session as Markdown in burrow-G's data folder.
var Export = command.Command{
	Name: "export",
	Help: "save the selected session as Markdown",
	Key:  "e",
	Run: func(env command.Env, in command.Input) (event.Event, error) {
		s, err := selected(env)
		if err != nil {
			return nil, err
		}
		return event.ExportRequested{Session: s}, nil
	},
}

func selected(env command.Env) (session.Session, error) {
	if env.Selected == nil {
		return session.Session{}, errNoSelection
	}
	s, ok := env.Selected()
	if !ok {
		return session.Session{}, errNoSelection
	}
	return s, nil
}
