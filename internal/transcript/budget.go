package transcript

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Budget caps a handoff prompt. Linux allows 128 KiB per command argument,
// and this leaves headroom while staying small next to agents' context windows.
const Budget = 100_000

const truncatedTail = "\n[truncated]\n\n"

func preamble(s session.Session) string {
	return fmt.Sprintf("You are continuing a coding session that started in %s (session %s, \"%s\") in %s.\n", s.Agent, s.ID, s.Title, s.Dir) +
		"Below is its transcript: user prompts, assistant replies, and one line per tool call (tool outputs are not included).\n" +
		"Read it, then briefly say where things stand and wait for my next instruction. Do not redo work that is already done.\n\n"
}

// Prompt is the preamble plus the masked transcript, trimmed to Budget.
// Redaction runs first, so the budget counts the masked text, and Masked
// counts only what was sent.
func Prompt(t session.Transcript) (string, Stats) {
	var r redactor
	head := cut(clean(r.mask(preamble(t.Session)+header(t, time.Time{}))), Budget/2)
	masked := r.count
	blocks := make([]string, len(t.Messages))
	counts := make([]int, len(t.Messages))
	for i, m := range t.Messages {
		var br redactor
		blocks[i] = clean(block(m, &br))
		counts[i] = br.count
	}
	body, kept := fit(blocks, firstUser(t.Messages), Budget-len(head))
	for _, i := range kept {
		masked += counts[i]
	}
	out := head + body
	return out, Stats{
		Messages: len(kept),
		Bytes:    len(out),
		Omitted:  len(blocks) - len(kept),
		Masked:   masked,
		Skipped:  t.Skipped,
	}
}

func firstUser(msgs []session.Message) int {
	for i, m := range msgs {
		if m.Role == "user" {
			return i
		}
	}
	return 0
}

// fit keeps the first user message and as many of the newest as fit in room,
// with one line standing in for the dropped middle. It returns the text and
// the indexes of the messages it kept.
func fit(blocks []string, first, room int) (string, []int) {
	total := 0
	for _, b := range blocks {
		total += len(b)
	}
	if total <= room {
		return strings.Join(blocks, ""), indexes(0, len(blocks)-1)
	}
	last := len(blocks) - 1
	if last == 0 {
		return cut(blocks[0], room), []int{0}
	}
	if first == last {
		first = 0
	}

	// The line's size depends on the count, so reserve the widest it can be.
	used := len(blocks[first]) + len(omittedLine(last))
	start := last + 1
	for i := last; i > first && used+len(blocks[i]) <= room; i-- {
		used += len(blocks[i])
		start = i
	}

	if start > last {
		// Even the first and newest message don't fit together: cut both.
		line := omittedLine(last - 1)
		if last-1 == 0 {
			line = ""
		}
		space := room - len(line)
		head := cut(blocks[first], min(len(blocks[first]), space/2))
		tail := cut(blocks[last], space-len(head))
		var kept []int
		if head != "" {
			kept = append(kept, first)
		}
		if tail != "" {
			kept = append(kept, last)
		}
		return head + line + tail, kept
	}

	kept := append([]int{first}, indexes(start, last)...)
	return blocks[first] + omittedLine(len(blocks)-len(kept)) + strings.Join(blocks[start:], ""), kept
}

func indexes(from, to int) []int {
	out := make([]int, 0, max(to-from+1, 0))
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func omittedLine(n int) string {
	return fmt.Sprintf("[%d messages omitted]\n\n", n)
}

// cut shortens s to at most n bytes on a character boundary, ending with [truncated].
func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	keep := n - len(truncatedTail)
	if keep <= 0 {
		return ""
	}
	for keep > 0 && !utf8.RuneStart(s[keep]) {
		keep--
	}
	return s[:keep] + truncatedTail
}
