package codex

// Export reads a Codex rollout file. Codex has no export command.
//
// A rollout is a plain list of lines, no tree: response_item lines are the
// conversation (OpenAI Responses API items), event_msg lines repeat them for
// the UI, turn_context names the model, compacted holds a summary. Items are
// grouped like every agent's export: one assistant record per reply, then
// one user record with that reply's tool results.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

type rolloutLine struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// item is one response_item payload; which fields are set depends on Type.
type item struct {
	Type      string          `json:"type"` // message, reasoning, function_call, …
	Role      string          `json:"role"`
	Content   []itemContent   `json:"content"`
	Summary   []itemContent   `json:"summary"`
	Name      string          `json:"name"`
	CallID    string          `json:"call_id"`
	Arguments string          `json:"arguments"` // function_call: JSON as text
	Input     string          `json:"input"`     // custom_tool_call: free text
	Action    json.RawMessage `json:"action"`    // local_shell_call, web_search_call
	Output    json.RawMessage `json:"output"`    // text, or {content, success}, or a list
}

type itemContent struct {
	Type     string `json:"type"` // input_text, output_text, input_image, summary_text
	Text     string `json:"text"`
	ImageURL string `json:"image_url"`
}

// injectedPrefixes start user messages Codex writes itself: the
// environment, AGENTS.md and user instructions.
var injectedPrefixes = []string{"<environment_context>", "<user_instructions>", "# AGENTS.md instructions"}

// rolloutFile finds the rollout whose session_meta has id. File names don't
// always carry the id, so each file's meta is read, as List does.
func (c Source) rolloutFile(id string) (string, error) {
	var found string
	err := filepath.WalkDir(filepath.Join(c.Home, "sessions"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" || d.IsDir() || !strings.HasPrefix(d.Name(), "rollout-") || !strings.HasSuffix(d.Name(), ".jsonl") {
			return err
		}
		if r, err := parseFile(path); err == nil && r.ID == id {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("codex session %q not found", id)
	}
	return found, nil
}

// header is the portable header for the rollout at path, titled like List.
func (c Source) header(path, id string) (portable.Header, error) {
	r, err := parseFile(path)
	if err != nil {
		return portable.Header{}, err
	}
	title := threadNames(filepath.Join(c.Home, "session_index.jsonl"))[id]
	if title == "" {
		title = helpers.Truncate(r.Prompt, 60, "…")
	}
	h := portable.Header{Agent: "codex", ID: id, Title: title, Dir: r.Cwd}
	err = scanRollout(path, func(l rolloutLine, ok bool) {
		if !ok || l.Timestamp.IsZero() {
			return
		}
		if h.Created.IsZero() {
			h.Created = l.Timestamp
		}
		h.Updated = l.Timestamp
	})
	return h, err
}

// scanRollout calls each for every line; ok is false for a line that isn't JSON.
func scanRollout(path string, each func(l rolloutLine, ok bool)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	for sc.Scan() {
		var l rolloutLine
		ok := json.Unmarshal(sc.Bytes(), &l) == nil
		each(l, ok)
	}
	return sc.Err()
}

// writeRollout writes the rollout's conversation as records.
func writeRollout(path string, w *portable.Writer) error {
	e := &emitter{w: w}
	err := scanRollout(path, func(l rolloutLine, ok bool) {
		if e.err != nil {
			return
		}
		switch {
		case !ok:
			w.Drop("unreadable line")
		case l.Type == "turn_context":
			var tc struct {
				Model string `json:"model"`
			}
			json.Unmarshal(l.Payload, &tc)
			e.model = tc.Model
			w.Drop("bookkeeping line")
		case l.Type == "compacted":
			var c struct {
				Message string `json:"message"`
			}
			json.Unmarshal(l.Payload, &c)
			e.flush()
			e.write(portable.Record{Kind: "compaction", Time: l.Timestamp, Text: c.Message})
		case l.Type == "response_item":
			var it item
			if json.Unmarshal(l.Payload, &it) != nil {
				w.Drop("unreadable line")
				return
			}
			e.item(it, l.Timestamp)
		default: // session_meta, event_msg (UI copies of items), …
			w.Drop("bookkeeping line")
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
	model   string           // from the latest turn_context
	reply   *portable.Record // assistant reply in progress
	results []portable.Part  // its tool results so far
	err     error
}

func (e *emitter) write(r portable.Record) {
	if e.err == nil {
		e.err = e.w.Record(r)
	}
}

// add puts an assistant side part on the reply. A part after tool results
// starts a new reply: the model answered those results.
func (e *emitter) add(p portable.Part, at time.Time) {
	if e.reply != nil && len(e.results) > 0 {
		e.flush()
	}
	if e.reply == nil {
		e.reply = &portable.Record{Kind: "message", Role: "assistant", Time: at, Model: e.model}
	}
	e.reply.Parts = append(e.reply.Parts, p)
}

func (e *emitter) flush() {
	if e.reply != nil {
		e.write(*e.reply)
	}
	if len(e.results) > 0 {
		at := time.Time{}
		if e.reply != nil {
			at = e.reply.Time
		}
		e.write(portable.Record{Kind: "message", Role: "user", Time: at, Parts: e.results})
	}
	e.reply, e.results = nil, nil
}

func (e *emitter) item(it item, at time.Time) {
	switch it.Type {
	case "message":
		e.message(it, at)
	case "reasoning":
		var text []string
		for _, s := range it.Summary {
			if s.Text != "" {
				text = append(text, s.Text)
			}
		}
		if len(text) == 0 { // only encrypted_content: nothing readable to carry
			e.w.Drop("thinking without text")
			return
		}
		e.add(portable.Part{Type: "reasoning", Text: strings.Join(text, "\n")}, at)
	case "function_call":
		e.add(portable.Part{Type: "tool_call", ID: it.CallID, Name: it.Name, Input: jsonOrString(it.Arguments)}, at)
	case "custom_tool_call":
		e.add(portable.Part{Type: "tool_call", ID: it.CallID, Name: it.Name, Input: jsonOrString(it.Input)}, at)
	case "local_shell_call":
		e.add(portable.Part{Type: "tool_call", ID: it.CallID, Name: "local_shell", Input: it.Action}, at)
	case "web_search_call":
		e.add(portable.Part{Type: "tool_call", ID: it.CallID, Name: "web_search", Input: it.Action}, at)
	case "function_call_output", "custom_tool_call_output", "local_shell_call_output":
		text, failed := output(it.Output)
		res := portable.Part{Type: "tool_result", ID: it.CallID, Text: text, Error: failed}
		if e.reply == nil {
			e.write(portable.Record{Kind: "message", Role: "user", Time: at, Parts: []portable.Part{res}})
			return
		}
		e.results = append(e.results, res)
	default:
		e.w.Drop("bookkeeping line")
	}
}

func (e *emitter) message(it item, at time.Time) {
	switch it.Role {
	case "assistant":
		for _, c := range it.Content {
			if c.Text != "" {
				e.add(portable.Part{Type: "text", Text: c.Text}, at)
			}
		}
	case "user":
		var parts []portable.Part
		for _, c := range it.Content {
			switch {
			case c.Type == "input_image":
				if img, ok := dataImage(c.ImageURL); ok {
					parts = append(parts, img)
				}
			case injected(c.Text):
				e.w.Drop("injected text")
			case strings.TrimSpace(c.Text) != "":
				parts = append(parts, portable.Part{Type: "text", Text: c.Text})
			}
		}
		if len(parts) > 0 {
			e.flush()
			e.write(portable.Record{Kind: "message", Role: "user", Time: at, Parts: parts})
		}
	default: // developer, system: instructions Codex sends, not the conversation
		e.w.Drop("injected text")
	}
}

func injected(text string) bool {
	t := strings.TrimSpace(text)
	for _, p := range injectedPrefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

// jsonOrString keeps arguments that are JSON as they are, and wraps free
// text as a JSON string, so Input is always valid JSON.
func jsonOrString(s string) json.RawMessage {
	if json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	b, _ := json.Marshal(s)
	return b
}

// output reads a tool output: plain text, {content, success}, or a list of
// content items.
func output(raw json.RawMessage) (text string, failed bool) {
	if json.Unmarshal(raw, &text) == nil {
		return text, false
	}
	var obj struct {
		Content string `json:"content"`
		Success *bool  `json:"success"`
	}
	if json.Unmarshal(raw, &obj) == nil && (obj.Content != "" || obj.Success != nil) {
		return obj.Content, obj.Success != nil && !*obj.Success
	}
	var list []itemContent
	json.Unmarshal(raw, &list)
	var b strings.Builder
	for _, c := range list {
		b.WriteString(c.Text)
	}
	return b.String(), false
}

// dataImage turns "data:image/png;base64,<data>" into an image part.
func dataImage(url string) (portable.Part, bool) {
	meta, data, ok := strings.Cut(strings.TrimPrefix(url, "data:"), ",")
	mime, enc, _ := strings.Cut(meta, ";")
	if !ok || !strings.HasPrefix(url, "data:") || enc != "base64" {
		return portable.Part{}, false
	}
	return portable.Part{Type: "image", MediaType: mime, Data: data}, true
}
