package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestRow(t *testing.T) {
	th := plainTheme()

	row := visible(Row(th, theme.Plain("Claude Code"), theme.Plain("59"), 20, false, false))
	if w := len([]rune(row)); w != 20 {
		t.Errorf("row width = %d, want 20: %q", w, row)
	}
	if !strings.HasPrefix(row, " Claude Code") || !strings.HasSuffix(row, "59 ") {
		t.Errorf("row = %q, want name left and count right", row)
	}
	if sel := Row(th, theme.Plain("Claude Code"), theme.Plain("59"), 20, true, true); !strings.Contains(sel, "▌") {
		t.Errorf("selected row should have the ▌ marker: %q", sel)
	}
	ascii := theme.Get(theme.Default, theme.Env{ASCII: true})
	if sel := Row(ascii, theme.Plain("x"), nil, 10, true, true); !strings.HasPrefix(visible(sel), ">") {
		t.Errorf("ASCII marker should be >: %q", sel)
	}
}

// The highlight must run through the whole row, also after a colored part
// like the star, whose color ends with a "reset everything" code.
func TestRow_HighlightCoversColoredParts(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	left := []theme.Part{{Text: "★", Style: th.Star}, {Text: " Title"}}
	row := Row(th, left, []theme.Part{{Text: "2 hours ago", Style: th.Muted}}, 40, true, true)

	before := row[:strings.Index(row, "Title")]
	afterReset := before[strings.LastIndex(before, "\x1b[0m"):]
	if !strings.Contains(afterReset, "48;") {
		t.Errorf("text after the ★ has no highlight background: %q", row)
	}
}

func TestSurfaceRow(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	row := SurfaceRow(th, theme.Plain("refresh"), theme.Plain("r"), 30, false)
	if w := ansi.StringWidth(row); w != 30 {
		t.Errorf("width = %d, want 30: %q", w, visible(row))
	}
	// Every cell carries the Surface background, also the gap and the padding.
	// Marker, text, gap, right text and padding each carry the background.
	if strings.Count(row, "48;") < 5 {
		t.Errorf("marker, text, gap and padding should all be on Surface: %q", row)
	}

	withProfile(t, theme.Color16.Profile())
	plain := theme.Get(theme.Default, theme.Env{Depth: theme.Color16})
	if row := SurfaceRow(plain, theme.Plain("refresh"), nil, 20, false); strings.Contains(row, "48;") || strings.Contains(row, "\x1b[4") {
		t.Errorf("16 colors should draw a plain row: %q", row)
	}
	if row := SurfaceRow(plain, theme.Plain("refresh"), nil, 20, true); !strings.Contains(row, "7") {
		t.Errorf("16 color selection should reverse: %q", row)
	}
}

func TestRow_Unfocused(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	focused := Row(th, theme.Plain("Codex"), nil, 20, true, true)
	unfocused := Row(th, theme.Plain("Codex"), nil, 20, true, false)
	if !strings.HasPrefix(visible(unfocused), "▌") {
		t.Errorf("an unfocused selection keeps its marker: %q", visible(unfocused))
	}
	if focused == unfocused {
		t.Error("unfocused selection should look dimmer than focused")
	}
	if !strings.Contains(unfocused, "48;") {
		t.Errorf("unfocused selection should have the SelectedInactive background: %q", unfocused)
	}
	if !strings.HasPrefix(unfocused, th.Muted.Render("▌")) {
		t.Errorf("unfocused marker should be Muted: %q", unfocused)
	}

	withProfile(t, theme.Color16.Profile())
	plain := theme.Get(theme.Default, theme.Env{Depth: theme.Color16})
	if row := Row(plain, theme.Plain("Codex"), nil, 20, true, false); strings.Contains(row, "\x1b[7m") || strings.Contains(row, ";7m") {
		t.Errorf("16 colors marks an unfocused selection with the marker only: %q", row)
	}
}
