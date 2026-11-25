package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestEmptyState(t *testing.T) {
	th := plainTheme()
	out := EmptyState(th, "no sessions yet", "start one in Codex and press r", 40, 5)
	lines := strings.Split(visible(out), "\n")
	if len(lines) != 5 || lipgloss.Width(out) != 40 {
		t.Fatalf("want a 40 × 5 area, got %d × %d: %q", lipgloss.Width(out), len(lines), lines)
	}
	if !strings.Contains(lines[1], "no sessions yet") || !strings.Contains(lines[2], "start one in Codex and press r") {
		t.Errorf("message and hint should sit in the middle: %q", lines)
	}
	if !strings.HasPrefix(lines[1], "    ") {
		t.Errorf("message should be centered: %q", lines[1])
	}

	if got := visible(EmptyState(th, "make the window bigger", "needs 40×8", 10, 1)); got != "make the …" {
		t.Errorf("one short line should cut the message and drop the hint: %q", got)
	}
	if EmptyState(th, "x", "y", 0, 3) != "" {
		t.Error("no room should draw nothing")
	}
}
