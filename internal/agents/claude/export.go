package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	var writeErr error
	err := scanLines(path, func(raw []byte) {
		if writeErr != nil {
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
			writeErr = writeLine(l, w)
		}
	})
	if err != nil {
		return err
	}
	return writeErr
}

// sideResult reports a tool result hanging off the live branch. With
// parallel tool calls Claude chains call A -> call B and hangs result A off
// call A as a side branch.
func sideResult(l exportLine, live map[string]bool) bool {
	return l.Type == "user" && live[l.parent()] && strings.Contains(string(l.Message), `"tool_result"`)
}

func writeLine(l exportLine, w *portable.Writer) error {
	var m exportMessage
	if json.Unmarshal(l.Message, &m) != nil {
		w.Drop("unreadable line")
		return nil
	}
	if l.IsCompactSummary {
		return w.Record(portable.Record{Kind: "compaction", Time: l.Timestamp, Text: text(m.Content)})
	}
	return w.Record(portable.Record{
		Kind: "message", Role: m.Role, Time: l.Timestamp, Model: m.Model,
		Parts: contentParts(m.Content),
	})
}

// contentBlock is one block of a message's content.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// contentParts turns content, a string or a list of blocks, into parts.
func contentParts(content json.RawMessage) []portable.Part {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return []portable.Part{{Type: "text", Text: s}}
	}
	var blocks []contentBlock
	json.Unmarshal(content, &blocks)
	var parts []portable.Part
	for _, b := range blocks {
		switch b.Type {
		case "text":
			parts = append(parts, portable.Part{Type: "text", Text: b.Text})
		case "tool_use":
			parts = append(parts, portable.Part{Type: "tool_call", ID: b.ID, Name: b.Name, Input: b.Input})
		case "tool_result":
			parts = append(parts, portable.Part{Type: "tool_result", ID: b.ToolUseID, Text: resultText(b.Content), Error: b.IsError})
		}
	}
	return parts
}

// resultText joins a tool result's text, which is a string or a list of blocks.
func resultText(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	var blocks []contentBlock
	json.Unmarshal(content, &blocks)
	var b strings.Builder
	for _, x := range blocks {
		if x.Type == "text" {
			b.WriteString(x.Text)
		}
	}
	return b.String()
}
