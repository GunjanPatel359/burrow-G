package theme

import (
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
