package handoff

// Drawing: one part of a record as Markdown, each kind cut to its limit
// (spec 0004, section 3) so no single item can crowd out the rest.

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

const (
	summaryLimit    = 40960 // the compaction summary
	textLimit       = 8192  // one message's text
	toolInputLimit  = 1024  // a tool call's arguments
	toolOutputLimit = 2048  // a tool's output
)

func drawSummary(text string) string { return cut(text, summaryLimit) }

func drawText(speaker, text string) string {
	return fmt.Sprintf("**%s:** %s", speaker, cut(text, textLimit))
}

// drawCall is the tool's name and its arguments. Arguments too long to show
// whole are cut, and cut JSON isn't JSON, so they become plain text.
func drawCall(p portable.Part) string {
	if len(p.Input) > toolInputLimit {
		return fmt.Sprintf("`%s` %s", p.Name, cut(string(p.Input), toolInputLimit))
	}
	return fmt.Sprintf("`%s` `%s`", p.Name, p.Input)
}

// drawResult is a tool's output in a fenced block, or an image's placeholder.
func drawResult(p portable.Part) string {
	if p.Type == "image" {
		return "[image]"
	}
	text := cut(p.Text, toolOutputLimit)
	fence := "```"
	for strings.Contains(text, fence) { // output with fences of its own
		fence += "`"
	}
	block := fence + "\n" + text + "\n" + fence
	if p.Error {
		return "failed:\n" + block
	}
	return block
}

// cut shortens s to at most limit bytes and says how much more there is. It
// cuts at a line break in the last fifth when there is one, so code stays
// readable, and never inside a character.
func cut(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	if nl := strings.LastIndexByte(s[:end], '\n'); nl > 0 && nl >= end-end/5 {
		end = nl
	}
	more := (len(s) - end + 1023) / 1024 // rounded up
	return s[:end] + fmt.Sprintf("\n… [%d KB more in the full record]", more)
}
