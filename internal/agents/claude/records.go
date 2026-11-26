package claude

// Export, pass 2: write the live lines as records.
//
// Records follow the API's shape, not the file's order: one assistant record
// per reply, then one user record with all of that reply's tool results.
// Claude writes a reply as several lines and can put results between them,
// so results wait until the reply ends.

import (
	"encoding/json"
	"slices"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

type exportMessage struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

func writeRecords(path string, live map[string]bool, w *portable.Writer) error {
	e := &emitter{w: w}
	err := scanLines(path, func(raw []byte) {
		if e.err != nil {
			return
		}
		var l exportLine
		if json.Unmarshal(raw, &l) != nil {
			w.Drop("unreadable line")
			return
		}
		switch {
		case l.IsSidechain:
			w.Drop("sidechain line")
		case !l.isMessage():
			w.Drop("bookkeeping line")
		case live[l.UUID] || sideResult(l, live):
			e.line(l)
		}
	})
	if err != nil {
		return err
	}
	e.flush()
	return e.err
}

type emitter struct {
	w       *portable.Writer
	reply   *portable.Record // assistant reply in progress
	replyID string           // its message.id
	results []portable.Part  // its tool results so far
	err     error            // first write error; later lines are skipped
}

func (e *emitter) write(r portable.Record) {
	if e.err == nil {
		e.err = e.w.Record(r)
	}
}

func (e *emitter) line(l exportLine) {
	var m exportMessage
	if json.Unmarshal(l.Message, &m) != nil {
		e.w.Drop("unreadable line")
		return
	}
	switch {
	case l.IsCompactSummary:
		e.flush()
		e.write(portable.Record{Kind: "compaction", Time: l.Timestamp, Text: text(m.Content)})

	case l.IsMeta:
		e.w.Drop("injected text")

	case l.Type == "assistant":
		parts := assistantParts(m.Content, e.w)
		if e.reply != nil && m.ID != "" && m.ID == e.replyID {
			e.reply.Parts = append(e.reply.Parts, parts...) // same reply, next line
			return
		}
		e.flush()
		e.reply = &portable.Record{Kind: "message", Role: "assistant", Time: l.Timestamp, Model: m.Model, Parts: parts}
		e.replyID = m.ID

	default: // user: tool results join the reply, words are a new turn
		results, said := userParts(m.Content, e.w)
		if len(results) > 0 {
			if e.reply != nil {
				e.results = append(e.results, results...)
			} else {
				e.write(portable.Record{Kind: "message", Role: "user", Time: l.Timestamp, Parts: results})
			}
		}
		if len(said) > 0 {
			e.flush()
			e.write(portable.Record{Kind: "message", Role: "user", Time: l.Timestamp, Parts: said})
		}
	}
}

// flush writes the reply in progress, then its results in the order of its
// tool calls. An image keeps its place after its result: the sort is stable
// and they share an id.
func (e *emitter) flush() {
	if e.reply == nil {
		return
	}
	e.write(*e.reply)
	if len(e.results) > 0 {
		order := map[string]int{}
		for i, p := range e.reply.Parts {
			if p.Type == "tool_call" {
				order[p.ID] = i
			}
		}
		pos := func(p portable.Part) int {
			if i, ok := order[p.ID]; ok {
				return i
			}
			return len(e.reply.Parts) // a result for no call here goes last
		}
		slices.SortStableFunc(e.results, func(a, b portable.Part) int { return pos(a) - pos(b) })
		e.write(portable.Record{Kind: "message", Role: "user", Time: e.reply.Time, Parts: e.results})
	}
	e.reply, e.replyID, e.results = nil, "", nil
}
