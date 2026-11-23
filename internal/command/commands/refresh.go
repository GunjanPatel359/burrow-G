package commands

import (
	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Refresh reloads the selected agent's sessions.
var Refresh = command.Command{
	Name: "refresh",
	Help: "reload the selected agent's sessions",
	Key:  "r",
	Run: func(command.Env, command.Input) (event.Event, error) {
		return event.Refresh{}, nil
	},
}
