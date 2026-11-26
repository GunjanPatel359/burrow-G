package claude

// Export, pass 1: find which lines of a session file are the conversation.
//
// A Claude session file is a tree, not a list: every line names its parent.
// Rewinds leave dead branches behind, each compaction starts a new tree, and
// parallel tool calls hang results off side branches. Pass 1 walks the tree
// once, keeping only uuid -> parent, and marks the live lines.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
