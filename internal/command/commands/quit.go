package commands

import (
	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Quit exits burrow-G.
var Quit = command.Command{
	Name: "quit",
	Help: "quit burrow-G",
	Key:  "q",
	Run: func(command.Env, command.Input) (event.Event, error) {
		return event.Quit{}, nil
	},
}
