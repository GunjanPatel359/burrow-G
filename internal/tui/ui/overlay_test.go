package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestOverlay(t *testing.T) {
	base := strings.Join([]string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccccccc", "dddddddddd", "eeeeeeeeee"}, "\n")
	got := visible(Overlay(base, "XX\nYY", 10, 5))
	want := strings.Join([]string{"aaaaaaaaaa", "bbbbXXbbbb", "ccccYYcccc", "dddddddddd", "eeeeeeeeee"}, "\n")
	if got != want {
		t.Errorf("Overlay =\n%s\nwant\n%s", got, want)
	}
}

func TestOverlay_ClampsToBase(t *testing.T) {
	base := "1234\n5678"
	got := Overlay(base, "ABCDEFG\nHIJKLMN\nOPQRSTU", 4, 2)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("overlay grew the base to %d lines", len(lines))
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w != 4 {
			t.Errorf("line %q is %d wide, want 4", visible(l), w)
		}
	}
}

func TestOverlay_ShortLinesAndStyles(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	base := th.Muted.Render("muted line here") + "\nab"
	got := Overlay(base, "XY", 10, 2)
	if lines := strings.Split(visible(got), "\n"); lines[0] != "muteXYline" || lines[1] != "ab" {
		t.Errorf("overlay = %q", lines)
	}
	if !strings.Contains(got, "\x1b[0mXY\x1b[0m") {
		t.Errorf("box should be fenced by resets so styles don't leak: %q", got)
	}
}
