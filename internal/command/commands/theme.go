package commands

import (
	"fmt"
	"slices"

	"github.com/GunjanPatel359/burrow-G/internal/command"
	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Theme changes the colour theme and saves the choice.
var Theme = command.Command{
	Name: "theme",
	Help: "change the colour theme",
	Args: []command.Arg{{
		Help:     "a colour theme",
		Required: true,
		Suggest:  suggestTheme,
		Check:    checkTheme,
	}},
	Run: func(env command.Env, in command.Input) (event.Event, error) {
		if err := env.Store.SetSetting("theme", in.Value()); err != nil {
			return nil, err
		}
		return event.ThemeChanged{Name: in.Value()}, nil
	},
}

// suggestTheme lists the themes, marks the saved one, and previews each row.
func suggestTheme(env command.Env, typed string) []command.Suggestion {
	saved := ""
	if env.Current != nil {
		saved = env.Current()
	}
	rows := make([]command.Suggestion, 0, len(env.Themes))
	for _, n := range env.Themes {
		row := command.Suggestion{Text: n, Preview: event.ThemeChanged{Name: n, Preview: true}}
		if n == saved {
			row.Hint = "✓"
		}
		rows = append(rows, row)
	}
	return command.Match(rows, typed)
}

func checkTheme(env command.Env, name string) error {
	if !slices.Contains(env.Themes, name) {
		return fmt.Errorf("no theme %q", name)
	}
	return nil
}
