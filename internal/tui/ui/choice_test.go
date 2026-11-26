package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestChoiceMarksSelected(t *testing.T) {
	th := plainTheme()
	box := visible(Choice(th, "Move session", []string{`"Fix it" · Claude Code`}, []string{"Save to file", "Continue in Codex"}, 1, nil, 80))
	lines := strings.Split(box, "\n")

	var save, codex string
	for _, l := range lines {
		switch {
		case strings.Contains(l, "Save to file"):
			save = l
		case strings.Contains(l, "Continue in Codex"):
			codex = l
		}
	}
	if !strings.Contains(codex, th.Glyph.Marker) || strings.Contains(save, th.Glyph.Marker) {
		t.Errorf("only the selected row should have the marker:\n%s", box)
	}
	if !strings.Contains(box, "Move session") || !strings.Contains(box, `"Fix it" · Claude Code`) {
		t.Errorf("title or lead missing:\n%s", box)
	}
	w := ansi.StringWidth(lines[0])
	for _, l := range lines {
		if ansi.StringWidth(l) != w || w > 50 || w < 24 {
			t.Errorf("box lines must all be %d cells, between 24 and 50: %q", w, l)
		}
	}
}
