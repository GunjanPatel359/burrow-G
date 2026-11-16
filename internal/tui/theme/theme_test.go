package theme

import (
	"slices"
	"strings"
	"testing"
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

	row := th.Row("Claude Code", "59", 20, false, false)
	if w := len([]rune(row)); w != 20 {
		t.Errorf("row width = %d, want 20: %q", w, row)
	}
	if !strings.HasPrefix(row, " Claude Code") || !strings.HasSuffix(row, "59 ") {
		t.Errorf("row = %q, want name left and count right", row)
	}

	if sel := th.Row("Claude Code", "59", 20, true, true); !strings.Contains(sel, "▌") {
		t.Errorf("selected row should have the ▌ marker: %q", sel)
	}
}
