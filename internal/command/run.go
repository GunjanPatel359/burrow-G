package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

// Run reads line, checks each argument and runs the command.
func (s *Service) Run(line string) (event.Event, error) {
	ws, open := split(line)
	if open {
		return nil, errors.New(`missing closing "`)
	}
	if len(ws) == 0 {
		return nil, errors.New("no command")
	}
	c, ok := s.find(ws[0].text)
	if !ok {
		return nil, fmt.Errorf("unknown command %q", ws[0].text)
	}

	// sort the words into @name:value tags and the bare value
	values := map[string]string{}
	var bare []string
	for _, w := range ws[1:] {
		if !w.isTag() {
			bare = append(bare, w.text)
			continue
		}
		name, value, ok := strings.Cut(w.text[1:], ":")
		if !ok || name == "" {
			return nil, fmt.Errorf("%q is not a tag (use @name:value)", w.text)
		}
		if c.arg(name) == nil {
			return nil, fmt.Errorf("%s doesn't take @%s", c.Name, name)
		}
		if _, dup := values[name]; dup {
			return nil, fmt.Errorf("@%s given twice", name)
		}
		values[name] = value
	}
	if len(bare) > 0 {
		if c.arg("") == nil {
			return nil, fmt.Errorf("%s takes no value", c.Name)
		}
		values[""] = strings.Join(bare, " ")
	}

	// every argument: present if required, and accepted by its Check
	for _, a := range c.Args {
		v, given := values[a.Name]
		switch {
		case !given && a.Required && a.Name == "":
			return nil, fmt.Errorf("%s needs a value: %s", c.Name, a.Help)
		case !given && a.Required:
			return nil, fmt.Errorf("%s needs @%s: %s", c.Name, a.Name, a.Help)
		case given && a.Check != nil:
			if err := a.Check(s.env, v); err != nil {
				return nil, err
			}
		}
	}
	return c.Run(s.env, Input{values})
}
