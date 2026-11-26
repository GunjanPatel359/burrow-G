package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
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

func targetNames(env command.Env) []string {
	lookPath := env.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	var names []string
	for _, src := range transcript.Targets(env.Sources, lookPath) {
		names = append(names, src.Name())
	}
	return names
}

func suggestTarget(env command.Env, typed string) []command.Suggestion {
	var rows []command.Suggestion
	for _, n := range targetNames(env) {
		rows = append(rows, command.Suggestion{Text: n})
	}
	return command.Match(rows, typed)
}

func checkTarget(env command.Env, name string) error {
	names := targetNames(env)
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
