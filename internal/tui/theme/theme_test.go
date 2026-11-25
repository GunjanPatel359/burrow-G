package theme

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestGet(t *testing.T) {
	if got := Get("tokyo-night", Env{}).Name; got != "tokyo-night" {
		t.Errorf("Get(tokyo-night).Name = %q", got)
	}
	if got := Get("no-such-theme", Env{}).Name; got != Default {
		t.Errorf("unknown theme should fall back to %q, got %q", Default, got)
	}
}

func TestNames(t *testing.T) {
	if !slices.Contains(Names(), Default) {
		t.Errorf("Names() = %v, missing %q", Names(), Default)
	}
}

func TestNames_AllThemes(t *testing.T) {
	want := []string{
		"ayu-mirage", "catppuccin-latte", "catppuccin-mocha", "dracula", "everforest-dark",
		"github-dark", "github-light", "gruvbox-dark", "kanagawa", "monokai", "nord", "one-dark",
		"rose-pine", "rose-pine-dawn", "solarized-dark", "solarized-light", "tokyo-night",
	}
	if got := Names(); !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
	for _, n := range want {
		if Get(n, Env{}).Name != n {
			t.Errorf("Get(%q).Name = %q", n, Get(n, Env{}).Name)
		}
	}
}

func TestGet_HasBackground(t *testing.T) {
	if got := Get("nord", Env{}).Background; got != "#2e3440" {
		t.Errorf("nord background = %q, want #2e3440", got)
	}
}

// With 256 colors the theme draws the hand picked indexes.
func TestGet_Color256(t *testing.T) {
	withProfile(t, termenv.ANSI256)
	th := Get("tokyo-night", Env{Depth: Color256})
	want := palettes["tokyo-night"]
	if got := th.Background; got != lipgloss.Color(strconv.Itoa(int(want.Background.X256))) {
		t.Errorf("background = %q, want index %d", got, want.Background.X256)
	}
	if got := th.Text.Render("x"); !strings.Contains(got, "38;5;"+strconv.Itoa(int(want.Text.X256))) {
		t.Errorf("text = %q, want 256 index %d", got, want.Text.X256)
	}
}

// At 16 colors every theme uses the shared map and keeps the terminal's background.
func TestGet_Color16(t *testing.T) {
	withProfile(t, Color16.Profile())
	th := Get("nord", Env{Depth: Color16})
	if th.Background != "" {
		t.Errorf("16 colors should not paint the background, got %q", th.Background)
	}
	if got := th.Error.Render("x"); !strings.Contains(got, "\x1b[31m") {
		t.Errorf("error = %q, want ANSI red", got)
	}
	if got := th.Selected.Render("x"); !strings.Contains(got, "7") || strings.Contains(got, "48;") {
		t.Errorf("selected = %q, want reverse and no background color", got)
	}
	if Get("dracula", Env{Depth: Color16}).Error.Render("x") != th.Error.Render("x") {
		t.Error("at 16 colors all themes should look the same")
	}
}

// Without color, attributes still mark muted text, selection and matches.
func TestGet_NoColor(t *testing.T) {
	withProfile(t, NoColor.Profile())
	th := Get(Default, Env{Depth: NoColor})
	if th.Background != "" {
		t.Errorf("no color should not paint the background, got %q", th.Background)
	}
	all := ""
	for _, s := range []lipgloss.Style{th.Header, th.Muted, th.Border, th.Text, th.Star, th.Folder,
		th.Label, th.Match, th.Selected, th.SelectedInactive, th.Surface, th.Marker, th.OK, th.Busy, th.Error} {
		all += s.Render("x")
	}
	if strings.Contains(all, "38;") || strings.Contains(all, "48;") || strings.Contains(all, "[3") {
		t.Errorf("no color drew a color: %q", all)
	}
	for name, s := range map[string]lipgloss.Style{"muted": th.Muted, "selected": th.Selected, "match": th.Match, "marker": th.Marker} {
		if s.Render("x") == "x" {
			t.Errorf("%s has no attribute without color", name)
		}
	}
}

func TestGet_ASCIIGlyphs(t *testing.T) {
	if g := Get(Default, Env{ASCII: true}).Glyph; g.Marker != ">" || g.Ellipsis != "..." {
		t.Errorf("ASCII env got %+v", g)
	}
	if g := Get(Default, Env{}).Glyph; g.Marker != "▌" {
		t.Errorf("Unicode env got marker %q", g.Marker)
	}
}

func TestGlyphSetsAreComplete(t *testing.T) {
	for name, g := range map[string]GlyphSet{"Unicode": Unicode, "ASCII": ASCII} {
		v := reflect.ValueOf(g)
		for i := range v.NumField() {
			if f := v.Field(i); f.Kind() == reflect.String && f.String() == "" {
				t.Errorf("%s.%s is empty", name, v.Type().Field(i).Name)
			}
		}
		if len(g.Spinner.Frames) == 0 || g.Spinner.FPS <= 0 || slices.Contains(g.Corners[:], "") {
			t.Errorf("%s spinner or corners incomplete: %+v", name, g)
		}
	}
}

func TestDepth(t *testing.T) {
	cases := []struct {
		profile  termenv.Profile
		depth    Depth
		renderer termenv.Profile
		paints   bool
	}{
		{termenv.TrueColor, TrueColor, termenv.TrueColor, true},
		{termenv.ANSI256, Color256, termenv.ANSI256, true},
		{termenv.ANSI, Color16, termenv.ANSI, false},
		{termenv.Ascii, NoColor, termenv.ANSI, false}, // Ascii would drop the attributes too
	}
	for _, c := range cases {
		d := DepthOf(c.profile)
		if d != c.depth || d.Profile() != c.renderer || d.Paints() != c.paints {
			t.Errorf("profile %v: depth %v, renderer %v, paints %v", c.profile, d, d.Profile(), d.Paints())
		}
	}
}

func TestASCIILocale(t *testing.T) {
	cases := []struct {
		env   map[string]string
		ascii bool
	}{
		{map[string]string{}, false},
		{map[string]string{"LANG": "en_US.UTF-8"}, false},
		{map[string]string{"LANG": "de_DE.utf8"}, false},
		{map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, true},
		{map[string]string{"LC_CTYPE": "POSIX"}, true},
		{map[string]string{"LC_ALL": "", "LC_CTYPE": "en_US.ISO-8859-1", "LANG": "en_US.UTF-8"}, true},
		{map[string]string{"LC_CTYPE": "en_US.UTF-8", "LANG": "C"}, false},
	}
	for _, c := range cases {
		getenv := func(k string) string { return c.env[k] }
		if got := DetectEnv(termenv.TrueColor, getenv).ASCII; got != c.ascii {
			t.Errorf("%v: ASCII = %v, want %v", c.env, got, c.ascii)
		}
	}
}

func TestInitial(t *testing.T) {
	never := func() bool { t.Error("dark() called although a known theme is saved"); return true }
	if got := Initial("nord", never); got != "nord" {
		t.Errorf("saved nord gave %q", got)
	}
	dark := func() bool { return true }
	light := func() bool { return false }
	for _, saved := range []string{"", "no-such-theme"} {
		if got := Initial(saved, dark); got != Default {
			t.Errorf("saved %q on dark gave %q, want %q", saved, got, Default)
		}
		if got := Initial(saved, light); got != DefaultLight {
			t.Errorf("saved %q on light gave %q, want %q", saved, got, DefaultLight)
		}
	}
}

func withProfile(t *testing.T, p termenv.Profile) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(p)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}
