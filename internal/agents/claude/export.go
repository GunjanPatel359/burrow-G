package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Claude has no structured export of its own (/export saves the terminal
// view), so Export reads the session's JSONL file itself.

// exportLine is the part of a Claude JSONL line export needs.
type exportLine struct {
	Type             string          `json:"type"`
	Subtype          string          `json:"subtype"`
	UUID             string          `json:"uuid"`
	ParentUUID       *string         `json:"parentUuid"`
	IsSidechain      bool            `json:"isSidechain"`
	IsCompactSummary bool            `json:"isCompactSummary"`
	IsMeta           bool            `json:"isMeta"`
	Timestamp        time.Time       `json:"timestamp"`
	LeafUUID         string          `json:"leafUuid"`
	Message          json.RawMessage `json:"message"`
}

func (l exportLine) parent() string {
	if l.ParentUUID == nil {
		return ""
	}
	return *l.ParentUUID
}

func (l exportLine) isMessage() bool { return l.Type == "user" || l.Type == "assistant" }

type exportMessage struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

type fileMeta struct {
	created, updated time.Time
}

func (c Source) Export(s session.Session, w *portable.Writer) error {
	path, err := c.sessionFile(s.ID)
	if err != nil {
		return err
	}
	title, cwd, err := parseFile(path)
	if err != nil {
		return err
	}
	live, meta, err := liveLines(path)
	if err != nil {
		return err
	}
	if err := w.Header(portable.Header{
		Agent: "claude", ID: s.ID, Title: title, Dir: cwd,
		Created: meta.created, Updated: meta.updated,
	}); err != nil {
		return err
	}
	return writeRecords(path, live, w)
}

// sessionFile finds projects/*/<id>.jsonl the way List found it. The id goes
// into a glob, so anything that could leave that folder is refused.
func (c Source) sessionFile(id string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\*?[`) {
		return "", fmt.Errorf("invalid claude session id %q", id)
	}
	matches, err := filepath.Glob(filepath.Join(c.Home, "projects", "*", id+".jsonl"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("claude session %q not found", id)
	}
	return matches[0], nil
}

func scanLines(path string, each func(raw []byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 64<<20) // one real Claude line is 860 KB
	for sc.Scan() {
		each(sc.Bytes())
	}
	return sc.Err()
}

// liveLines is pass 1: it keeps only uuid -> parent and finds the lines on
// the conversation's live branch, in every compaction segment.
func liveLines(path string) (map[string]bool, fileMeta, error) {
	parent := map[string]string{}
	var meta fileMeta
	var leaf, lastMsg string
	var segLeaves []string

	err := scanLines(path, func(raw []byte) {
		var l exportLine
		if json.Unmarshal(raw, &l) != nil {
			return
		}
		if !l.Timestamp.IsZero() {
			if meta.created.IsZero() {
				meta.created = l.Timestamp
			}
			meta.updated = l.Timestamp
		}
		if l.Type == "last-prompt" && l.LeafUUID != "" {
			leaf = l.LeafUUID
		}
		if l.UUID == "" {
			return
		}
		parent[l.UUID] = l.parent()
		// A compact boundary starts a new chain. Its logicalParentUuid points
		// forward, not back, so the older segment ends at its last message.
		if l.Type == "system" && l.parent() == "" && lastMsg != "" {
			segLeaves = append(segLeaves, lastMsg)
		}
		if l.isMessage() && !l.IsSidechain {
			lastMsg = l.UUID
		}
	})
	if err != nil {
		return nil, meta, err
	}

	if _, ok := parent[leaf]; !ok {
		leaf = lastMsg
	}
	live := map[string]bool{}
	for _, lf := range append(segLeaves, leaf) {
		for u := lf; u != "" && !live[u]; u = parent[u] {
			live[u] = true
		}
	}
	return live, meta, nil
}

// writeRecords is pass 2: it streams the live lines as records and counts
// what it leaves out.
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

// sideResult reports a tool result hanging off the live branch. With
// parallel tool calls Claude chains call A -> call B and hangs result A off
// call A as a side branch.
func sideResult(l exportLine, live map[string]bool) bool {
	return l.Type == "user" && live[l.parent()] && strings.Contains(string(l.Message), `"tool_result"`)
}

// emitter groups lines into records the way the API sees a conversation:
// one assistant record per reply, then one user record with all of that
// reply's tool results. Claude writes a reply as several lines and can put
// results between them, so results wait until the reply ends.
type emitter struct {
	w       *portable.Writer
	cur     *portable.Record // reply in progress
	curID   string           // its message.id
	results []portable.Part  // its tool results so far
	err     error
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
		parts := e.parts(m.Content)
		if e.cur != nil && m.ID != "" && m.ID == e.curID {
			e.cur.Parts = append(e.cur.Parts, parts...)
			return
		}
		e.flush()
		e.cur = &portable.Record{Kind: "message", Role: "assistant", Time: l.Timestamp, Model: m.Model, Parts: parts}
		e.curID = m.ID
	default:
		var results, said []portable.Part
		for _, p := range e.parts(m.Content) {
			if p.Type == "tool_result" || (p.Type == "image" && p.ID != "") {
				results = append(results, p)
			} else {
				said = append(said, p)
			}
		}
		if len(results) > 0 {
			if e.cur != nil {
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
	if e.cur == nil {
		return
	}
	e.write(*e.cur)
	if len(e.results) > 0 {
		order := map[string]int{}
		for i, p := range e.cur.Parts {
			if p.Type == "tool_call" {
				order[p.ID] = i
			}
		}
		pos := func(p portable.Part) int {
			if i, ok := order[p.ID]; ok {
				return i
			}
			return len(e.cur.Parts) // a result for no call here goes last
		}
		slices.SortStableFunc(e.results, func(a, b portable.Part) int { return pos(a) - pos(b) })
		e.write(portable.Record{Kind: "message", Role: "user", Time: e.cur.Time, Parts: e.results})
	}
	e.cur, e.curID, e.results = nil, "", nil
}

// contentBlock is one block of a message's content.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
	Source    struct {
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source"`
}

// parts turns content, a string or a list of blocks, into parts, counting
// what it leaves out.
func (e *emitter) parts(content json.RawMessage) []portable.Part {
	var s string
	if json.Unmarshal(content, &s) == nil {
		if s = e.userText(s); s == "" {
			return nil
		}
		return []portable.Part{{Type: "text", Text: s}}
	}
	var blocks []contentBlock
	json.Unmarshal(content, &blocks)
	var parts []portable.Part
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := e.userText(b.Text); t != "" {
				parts = append(parts, portable.Part{Type: "text", Text: t})
			}
		case "thinking":
			// Claude often keeps only the signature; there's nothing to carry.
			if b.Thinking == "" {
				e.w.Drop("thinking without text")
				continue
			}
			parts = append(parts, portable.Part{Type: "reasoning", Text: b.Thinking})
		case "tool_use":
			parts = append(parts, portable.Part{Type: "tool_call", ID: b.ID, Name: b.Name, Input: b.Input})
		case "tool_result":
			parts = append(parts, e.result(b)...)
		case "image":
			parts = append(parts, portable.Part{Type: "image", MediaType: b.Source.MediaType, Data: b.Source.Data})
		}
	}
	return parts
}

// result is a tool result part, then one image part per image in it, with
// the result's id so an importer can put them back inside the result.
func (e *emitter) result(b contentBlock) []portable.Part {
	res := portable.Part{Type: "tool_result", ID: b.ToolUseID, Error: b.IsError}
	var images []portable.Part
	var s string
	if json.Unmarshal(b.Content, &s) == nil {
		res.Text = s
	} else {
		var inner []contentBlock
		json.Unmarshal(b.Content, &inner)
		var t strings.Builder
		for _, x := range inner {
			switch x.Type {
			case "text":
				t.WriteString(x.Text)
			case "image":
				images = append(images, portable.Part{Type: "image", ID: b.ToolUseID, MediaType: x.Source.MediaType, Data: x.Source.Data})
			}
		}
		res.Text = t.String()
	}
	return append([]portable.Part{res}, images...)
}

// injectedTags are blocks Claude Code adds to a user turn; the user never
// typed them.
var injectedTags = []string{"system-reminder", "local-command-caveat", "local-command-stdout", "local-command-stderr"}

// userText removes injected blocks from text and turns a slash command into
// what the user typed. Assistant text has no such blocks, so it passes
// through unchanged.
func (e *emitter) userText(s string) string {
	if name := helpers.Between(s, "<command-name>", "</command-name>"); name != "" {
		args := helpers.Between(s, "<command-args>", "</command-args>")
		return strings.TrimSpace(name + " " + args)
	}
	for _, tag := range injectedTags {
		open, close := "<"+tag+">", "</"+tag+">"
		for {
			start := strings.Index(s, open)
			if start < 0 {
				break
			}
			end := strings.Index(s[start:], close)
			if end < 0 {
				s = s[:start] // unterminated: the rest is the block
			} else {
				s = s[:start] + s[start+end+len(close):]
			}
			e.w.Drop("injected text")
		}
	}
	return strings.TrimSpace(s)
}
