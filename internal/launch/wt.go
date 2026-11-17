package launch

import "strings"

type windowsTerminal struct {
	run      func(name string, args ...string) error
	lookPath func(file string) (string, error)
}

func (windowsTerminal) Name() string { return "windows-terminal" }

// OpenTab passes the agent's full path, so npm's .cmd shims (claude.cmd) are found.
func (w windowsTerminal) OpenTab(dir, title string, argv []string) error {
	path, err := w.lookPath(argv[0])
	if err != nil {
		return err
	}
	argv = append([]string{path}, argv[1:]...)
	return w.run("wt.exe", wtArgs(dir, title, argv)...)
}

// wtArgs opens a new tab in the current window. wt reads ";" as "next command", so it is escaped.
func wtArgs(dir, title string, argv []string) []string {
	args := append([]string{"-w", "0", "new-tab", "--title", title, "-d", dir}, argv...)
	for i, a := range args {
		args[i] = strings.ReplaceAll(a, ";", `\;`)
	}
	return args
}
