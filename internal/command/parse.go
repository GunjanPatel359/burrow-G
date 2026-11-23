package command

import "strings"

// word is one word of a line; quoted says it began with a quote,
// start is its byte offset in the line.
type word struct {
	text   string
	quoted bool
	start  int
}

// isTag reports whether w is written as @… (a quoted "@…" is plain text).
func (w word) isTag() bool { return !w.quoted && strings.HasPrefix(w.text, "@") }

// split breaks line on spaces outside quotes and drops the quotes.
// open reports an unclosed quote.
func split(line string) (ws []word, open bool) {
	var b strings.Builder
	var cur word
	started := false
	for i, r := range line {
		if !started && (r != ' ' || open) {
			cur.start = i
		}
		switch {
		case r == '"':
			if !started {
				cur.quoted = true
			}
			open = !open
			started = true
		case r == ' ' && !open:
			if started {
				cur.text = b.String()
				ws = append(ws, cur)
			}
			b.Reset()
			cur, started = word{}, false
		default:
			b.WriteRune(r)
			started = true
		}
	}
	if started {
		cur.text = b.String()
		ws = append(ws, cur)
	}
	return ws, open
}
