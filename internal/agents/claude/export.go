package claude

// Export reads a Claude session file in two passes. Pass 1 finds which
// lines are the conversation; pass 2 writes them as records; the last
// section turns a message's content into parts.

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
)

// ---- Pass 1: find which lines of a session file are the conversation.
//
// A Claude session file is a tree, not a list: every line names its parent.
// Rewinds leave dead branches behind, each compaction starts a new tree, and
// parallel tool calls hang results off side branches. Pass 1 walks the tree
// once, keeping only uuid -> parent, and marks the live lines.

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

// fileMeta is what the header needs from the whole file.
type fileMeta struct {
	created, updated time.Time
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

// scanLines calls each for every line of path.
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

// liveLines marks the lines on the live branch of every compaction segment.
func liveLines(path string) (map[string]bool, fileMeta, error) {
	parent := map[string]string{}
	var meta fileMeta
	var leaf, lastMsg string // leaf: Claude's own pointer; lastMsg: the fallback
	var segLeaves []string   // where each older segment ends

	err := scanLines(path, func(raw []byte) {
		var l exportLine
		if json.Unmarshal(raw, &l) != nil {
			return // pass 2 counts it
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
		// A compact boundary starts a new tree. Its logicalParentUuid points
		// forward, not back, so the older segment ends at its last message.
		if l.Subtype == "compact_boundary" && lastMsg != "" {
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

// sideResult reports a tool result hanging off the live branch. With
// parallel tool calls Claude chains call A -> call B and hangs result A off
// call A as a side branch.
func sideResult(l exportLine, live map[string]bool) bool {
	return l.Type == "user" && live[l.parent()] && strings.Contains(string(l.Message), `"tool_result"`)
}

// ---- Pass 2: write the live lines as records.
//
// Records follow the API's shape, not the file's order: one assistant record
// per reply, then one user record with all of that reply's tool results.
// Claude writes a reply as several lines and can put results between them,
// so results wait until the reply ends.

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
		e.write(portable.Record{Kind: "compaction", Time: l.Timestamp, Text: withoutInjected(text(m.Content), e.w)})

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

// ---- Content: turn a message's content into portable parts.
//
// Content is a string or a list of blocks. User turns also carry blocks
// Claude Code injected (reminders, local command output), which are removed;
// assistant text is kept word for word.

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

func blocks(content json.RawMessage) []contentBlock {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return []contentBlock{{Type: "text", Text: s}}
	}
	var list []contentBlock
	json.Unmarshal(content, &list)
	return list
}

func assistantParts(content json.RawMessage, w *portable.Writer) []portable.Part {
	var parts []portable.Part
	for _, b := range blocks(content) {
		switch b.Type {
		case "text":
			parts = append(parts, portable.Part{Type: "text", Text: b.Text})
		case "thinking":
			// Claude often keeps only the signature; there's nothing to carry.
			if b.Thinking == "" {
				w.Drop("thinking without text")
				continue
			}
			parts = append(parts, portable.Part{Type: "reasoning", Text: b.Thinking})
		case "tool_use":
			parts = append(parts, portable.Part{Type: "tool_call", ID: b.ID, Name: b.Name, Input: b.Input})
		case "image":
			parts = append(parts, image(b, ""))
		}
	}
	return parts
}

// userParts splits a user turn into tool results (with their images) and
// what the user said.
func userParts(content json.RawMessage, w *portable.Writer) (results, said []portable.Part) {
	for _, b := range blocks(content) {
		switch b.Type {
		case "text":
			if t := userText(b.Text, w); t != "" {
				said = append(said, portable.Part{Type: "text", Text: t})
			}
		case "image":
			said = append(said, image(b, ""))
		case "tool_result":
			results = append(results, toolResult(b)...)
		}
	}
	return results, said
}

// toolResult is the result part, then one image part per image in it with
// the result's id, so an importer can put them back inside the result.
func toolResult(b contentBlock) []portable.Part {
	res := portable.Part{Type: "tool_result", ID: b.ToolUseID, Error: b.IsError}
	var images []portable.Part
	var text strings.Builder
	for _, x := range blocks(b.Content) {
		switch x.Type {
		case "text":
			text.WriteString(x.Text)
		case "image":
			images = append(images, image(x, b.ToolUseID))
		}
	}
	res.Text = text.String()
	return append([]portable.Part{res}, images...)
}

func image(b contentBlock, id string) portable.Part {
	return portable.Part{Type: "image", ID: id, MediaType: b.Source.MediaType, Data: b.Source.Data}
}

// injectedTags are blocks Claude Code adds to a user turn; the user never
// typed them.
var injectedTags = []string{"system-reminder", "local-command-caveat", "local-command-stdout", "local-command-stderr"}

// userText removes injected blocks and turns a slash command back into what
// the user typed.
func userText(s string, w *portable.Writer) string {
	if name := helpers.Between(s, "<command-name>", "</command-name>"); name != "" {
		args := helpers.Between(s, "<command-args>", "</command-args>")
		return strings.TrimSpace(name + " " + args)
	}
	return withoutInjected(s, w)
}

// withoutInjected removes the blocks the agent injected, counting each.
// Compaction summaries carry them too.
func withoutInjected(s string, w *portable.Writer) string {
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
			w.Drop("injected text")
		}
	}
	return strings.TrimSpace(s)
}
