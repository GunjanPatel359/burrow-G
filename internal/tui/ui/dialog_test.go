package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var (
	yes = key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "delete"))
	no  = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel"))
)

func TestDialog(t *testing.T) {
	th := plainTheme()
	body := []string{`"Fix the flaky login test that fails on CI every other run"`, "~/work/app"}
	box := Dialog(th, "delete session", body, []key.Binding{yes, no}, 80)
	lines := strings.Split(box, "\n")

	for i, l := range lines {
		if w := ansi.StringWidth(l); w != 50 {
			t.Errorf("line %d is %d wide, want 50: %q", i, w, visible(l))
		}
	}
	text := visible(box)
	if !strings.HasPrefix(lines[0], "╭") || !strings.HasPrefix(visible(lines[len(lines)-1]), "╰") {
		t.Errorf("no rounded frame:\n%s", text)
	}
	for _, want := range []string{"delete session", "flaky login", "every other run", "~/work/app", "y delete · n cancel"} {
		if !strings.Contains(text, want) {
			t.Errorf("dialog lost %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "…") {
		t.Errorf("the body must wrap, never cut:\n%s", text)
	}
}

func TestDialog_Width(t *testing.T) {
	th := plainTheme()
	for _, c := range []struct{ body, want int }{{200, 50}, {40, 36}, {10, 24}} {
		first := strings.Split(Dialog(th, "t", nil, nil, c.body), "\n")[0]
		if w := ansi.StringWidth(first); w != c.want {
			t.Errorf("body width %d: box %d wide, want %d", c.body, w, c.want)
		}
	}
}

func TestDialog_FilledWithSurface(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	row := strings.Split(Dialog(th, "delete session", []string{"x"}, []key.Binding{yes}, 80), "\n")[1]
	surface := palette(t, th)
	if strings.Count(row, surface) < 3 {
		t.Errorf("title row is not filled with Surface %q: %q", surface, row)
	}

	withProfile(t, theme.NoColor.Profile())
	plain := theme.Get(theme.Default, theme.Env{Depth: theme.NoColor})
	if box := Dialog(plain, "delete session", []string{"x"}, []key.Binding{yes}, 80); !strings.Contains(box, "╭") || strings.Contains(box, "48;") {
		t.Errorf("without color the dialog keeps its frame and no fill: %q", box)
	}
}

// palette is the escape code fragment of the theme's Surface background.
func palette(t *testing.T, th theme.Theme) string {
	t.Helper()
	s := th.Surface.Render(" ")
	i := strings.Index(s, "48;")
	if i < 0 {
		t.Fatalf("Surface has no background: %q", s)
	}
	return s[i : strings.IndexByte(s[i:], 'm')+i]
}
