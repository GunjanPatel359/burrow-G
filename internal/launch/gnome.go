package launch

type gnome struct {
	run func(name string, args ...string) error
}

func (gnome) Name() string { return "gnome-terminal" }

func (g gnome) OpenTab(dir, title string, argv []string) error {
	return g.run("gnome-terminal", gnomeArgs(dir, title, argv)...)
}

func gnomeArgs(dir, title string, argv []string) []string {
	args := []string{"--tab", "--title=" + title, "--working-directory=" + dir, "--"}
	return append(args, argv...)
}
