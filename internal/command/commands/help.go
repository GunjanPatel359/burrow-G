package commands

import (
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Help lists the keys and every command. It needs the list of commands, so
// New builds it with a way to read that list.
func Help(list func() []command.Command) command.Command {
	return command.Command{
		Name: "help",
		Help: "show keys and commands",
		Run: func(env command.Env, _ command.Input) (event.Event, error) {
			var b strings.Builder
			b.WriteString("Keys\n")
			for _, k := range env.Keys {
				b.WriteString(k + "\n")
			}
			b.WriteString("\nCommands\n")
			for _, c := range list() {
				b.WriteString(command.Usage(c) + "  " + c.Help + "\n")
			}
			return event.ShowHelp{Text: b.String()}, nil
		},
	}
}
