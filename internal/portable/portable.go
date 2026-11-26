// Package portable is the agent neutral file format for an exported session
// (spec 0003): one JSON object per line, a header, the records, an end line.
package portable

import (
	"encoding/json"
	"errors"
	"time"
)

// Format names this version. A change that breaks old readers bumps it.
const Format = "burrow-session/1"

// ErrTruncated means the file stopped before its end line, so it is not a
// whole session.
var ErrTruncated = errors.New("export file is cut off: no end line")

// maxLine is the longest line a Reader accepts; one real Claude line is 860 KB.
const maxLine = 64 << 20

type Header struct {
	Kind     string    `json:"kind"`   // "header"
	Format   string    `json:"format"` // Format
	Agent    string    `json:"agent"`  // Source.Name(): tool names below are this agent's
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Dir      string    `json:"dir"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	Exported time.Time `json:"exported"`
	Burrow   string    `json:"burrow"` // burrow-G version that wrote it
}

type Record struct {
	Kind  string    `json:"kind"`           // "message" or "compaction"
	Role  string    `json:"role,omitempty"` // message: "user" or "assistant"
	Time  time.Time `json:"time"`
	Model string    `json:"model,omitempty"` // assistant: the model id the agent recorded
	Parts []Part    `json:"parts,omitempty"` // message
	Text  string    `json:"text,omitempty"`  // compaction: the summary
}

type Part struct {
	Type      string          `json:"type"`                 // text, reasoning, tool_call, tool_result, image
	Text      string          `json:"text,omitempty"`       // text, reasoning, tool_result
	ID        string          `json:"id,omitempty"`         // tool_call and its tool_result share it
	Name      string          `json:"name,omitempty"`       // tool_call: the agent's own tool name
	Input     json.RawMessage `json:"input,omitempty"`      // tool_call: the agent's own arguments
	Error     bool            `json:"error,omitempty"`      // tool_result: the tool failed
	MediaType string          `json:"media_type,omitempty"` // image
	Data      string          `json:"data,omitempty"`       // image: base64
}

type End struct {
	Kind    string         `json:"kind"` // "end"
	Records int            `json:"records"`
	Dropped map[string]int `json:"dropped,omitempty"` // what was left out, by reason
}
