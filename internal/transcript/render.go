// Package transcript turns a session's conversation into Markdown, for export
// and for handing it off to another agent as a first prompt.
package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Stats counts what an export or handoff carried.
type Stats struct {
	Messages int // messages included
	Bytes    int // size of the output
	Omitted  int // messages dropped to fit the budget
	Masked   int // secrets replaced with [redacted]
	Skipped  int // entries the reader could not parse
}

// CheckMessages fails when nothing in the session could be read.
func CheckMessages(t session.Transcript) error {
	if len(t.Messages) == 0 {
		return fmt.Errorf("no messages found in %s session %s", t.Session.Agent, t.Session.ID)
	}
	return nil
}

// Markdown renders the whole transcript, masked but never trimmed.
func Markdown(t session.Transcript, now time.Time) (string, Stats) {
	var r redactor
	var b strings.Builder
	b.WriteString(r.mask(header(t, now)))
	for _, m := range t.Messages {
		b.WriteString(r.mask(block(m)))
	}
	out := b.String()
	return out, Stats{Messages: len(t.Messages), Bytes: len(out), Masked: r.count, Skipped: t.Skipped}
}

// header is the title, the session facts, and the skipped count when there is one.
// A zero now leaves out the export time.
func header(t session.Transcript, now time.Time) string {
	s := t.Session
	facts := []string{"Agent: " + s.Agent, "Session: " + s.ID, "Folder: " + s.Dir}
	if !now.IsZero() {
		facts = append(facts, "Exported: "+now.Format(time.RFC3339))
	}
	h := "# " + s.Title + "\n\n" + strings.Join(facts, " · ") + "\n"
	if t.Skipped > 0 {
		h += fmt.Sprintf("(%d entries skipped)\n", t.Skipped)
	}
	return h + "\n"
}

func block(m session.Message) string {
	role := "User"
	if m.Role == "assistant" {
		role = "Assistant"
	}
	var b strings.Builder
	b.WriteString("## " + role)
	if !m.Time.IsZero() {
		b.WriteString(" · " + m.Time.Local().Format("2006-01-02 15:04"))
	}
	b.WriteString("\n\n")
	if m.Text != "" {
		b.WriteString(strings.TrimSpace(m.Text) + "\n\n")
	}
	if len(m.Tools) > 0 {
		for _, tool := range m.Tools {
			b.WriteString("- tool: " + tool + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

var primaryArgs = []string{"file_path", "filePath", "path", "command", "cmd", "pattern", "url", "query"}

// ToolLine is "<name> <primary argument>". input is the tool's JSON input,
// either an object or a string holding one (Codex and Copilot quote it).
func ToolLine(name string, input json.RawMessage) string {
	var quoted string
	if json.Unmarshal(input, &quoted) == nil {
		input = json.RawMessage(quoted)
	}
	var args map[string]any
	if json.Unmarshal(input, &args) != nil {
		return name
	}
	for _, key := range primaryArgs {
		v, ok := args[key]
		if !ok {
			continue
		}
		arg := argString(v)
		if arg == "" {
			continue
		}
		return name + " " + helpers.Truncate(arg, 120, "…")
	}
	return name
}

// argString reads a value as text; commands can come as a list of words.
func argString(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case []any:
		words := make([]string, 0, len(v))
		for _, w := range v {
			if s, ok := w.(string); ok {
				words = append(words, s)
			}
		}
		return strings.Join(words, " ")
	}
	return ""
}
