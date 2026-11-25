package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// reset ends any style a cut line left open, so it can't leak into the box
// or out of it.
const reset = "\x1b[0m"

// Overlay draws box centered over the first height lines of base, each width
// cells wide. A box bigger than that is cut to fit; base never grows.
func Overlay(base, box string, width, height int) string {
	lines := strings.Split(base, "\n")
	height = min(height, len(lines))
	boxLines := strings.Split(box, "\n")
	boxLines = boxLines[:min(len(boxLines), height)]

	boxWidth := 0
	for _, l := range boxLines {
		boxWidth = max(boxWidth, ansi.StringWidth(l))
	}
	boxWidth = min(boxWidth, width)
	top, left := (height-len(boxLines))/2, (width-boxWidth)/2

	for i, b := range boxLines {
		row := lines[top+i]
		if w := ansi.StringWidth(row); w < width {
			row += strings.Repeat(" ", width-w)
		}
		b = ansi.Truncate(b, boxWidth, "")
		b += strings.Repeat(" ", boxWidth-ansi.StringWidth(b))
		lines[top+i] = ansi.Cut(row, 0, left) + reset + b + reset + ansi.Cut(row, left+boxWidth, width)
	}
	return strings.Join(lines, "\n")
}
