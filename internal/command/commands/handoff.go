package commands

import (
	"fmt"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Handoff continues the selected session in another agent. Without
// @agent: the TUI opens a picker.
var Handoff = command.Command{
	Name: "handoff",
	Help: "continue the selected session in another agent",
	Key:  "t",
	Args: []command.Arg{{
		Name:    "agent",
		Help:    "the agent to continue in",
		Suggest: suggestTarget,
		Check:   checkTarget,
	}},
	Run: func(env command.Env, in command.Input) (event.Event, error) {
		s, err := selected(env)
		if err != nil {
			return nil, err
		}
		return event.HandoffRequested{Session: s, Agent: in.Get("agent")}, nil
	},
}

func targetNames(env command.Env) ([]string, bool) {
	if env.Targets == nil {
		return nil, true
	}
	return env.Targets()
}

func suggestTarget(env command.Env, typed string) []command.Suggestion {
	names, _ := targetNames(env)
	var rows []command.Suggestion
	for _, n := range names {
		rows = append(rows, command.Suggestion{Text: n})
	}
	return command.Match(rows, typed)
}

// checkTarget accepts any name before the lookup has answered; the handoff
// itself checks PATH again, so nothing wrong can launch.
func checkTarget(env command.Env, name string) error {
	names, loaded := targetNames(env)
	if !loaded {
		return nil
	}
	for _, n := range names {
		if n == name {
			return nil
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("no agent CLI found on PATH")
	}
	return fmt.Errorf("no installed agent %q (choose from: %s)", name, strings.Join(names, ", "))
}
