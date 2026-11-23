package theme

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestGet(t *testing.T) {
	if got := Get("tokyo-night").Name; got != "tokyo-night" {
		t.Errorf("Get(tokyo-night).Name = %q", got)
	}
	if got := Get("no-such-theme").Name; got != Default {
		t.Errorf("unknown theme should fall back to %q, got %q", Default, got)
	}
}

func TestNames(t *testing.T) {
	if !slices.Contains(Names(), Default) {
		t.Errorf("Names() = %v, missing %q", Names(), Default)
	}
}

func TestRow(t *testing.T) {
	th := Get(Default)

	row := th.Row(Plain("Claude Code"), Plain("59"), 20, false, false)
	if w := len([]rune(row)); w != 20 {
		t.Errorf("row width = %d, want 20: %q", w, row)
	}
	if !strings.HasPrefix(row, " Claude Code") || !strings.HasSuffix(row, "59 ") {
		t.Errorf("row = %q, want name left and count right", row)
	}

	if sel := th.Row(Plain("Claude Code"), Plain("59"), 20, true, true); !strings.Contains(sel, "▌") {
		t.Errorf("selected row should have the ▌ marker: %q", sel)
	}
}

// The highlight must run through the whole row, also after a coloured part
// like the ★, whose colour ends with a "reset everything" code.
func TestRow_HighlightCoversColouredParts(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })

	th := Get(Default)
	left := []Part{{Text: "★", Style: th.Star}, {Text: " Title"}}
	row := th.Row(left, []Part{{Text: "2 hours ago", Style: th.Muted}}, 40, true, true)

	before := row[:strings.Index(row, "Title")]
	afterReset := before[strings.LastIndex(before, "\x1b[0m"):]
	if !strings.Contains(afterReset, "48;") {
		t.Errorf("text after the ★ has no highlight background: %q", row)
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
		if Get(n).Name != n {
			t.Errorf("Get(%q).Name = %q", n, Get(n).Name)
		}
	}
}

func TestEveryPaletteIsComplete(t *testing.T) {
	for name, p := range palettes {
		v := reflect.ValueOf(p)
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s: %s is empty", name, v.Type().Field(i).Name)
			}
		}
	}
}

func TestGet_HasBackground(t *testing.T) {
	if got := Get("nord").Background; got != "#2e3440" {
		t.Errorf("nord background = %q, want #2e3440", got)
	}
}
