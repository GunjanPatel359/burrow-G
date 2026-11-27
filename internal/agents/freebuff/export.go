package freebuff

// Export reads a chat's chat-messages.json: the whole conversation as
// Freebuff shows it, which keeps the earlier messages after a compaction.
// The compaction summary is only in run-state.json, as the first message of
// the history the model sees; it goes in where it happened. Freebuff keeps
// only the latest one, so an earlier summary can't be exported.
//
// One AI message is a whole turn: text, tool calls, more text. It is split
// into replies the way the model made them: tool calls join the reply of
// the text before them, and text after a tool call starts the next reply.
// Each reply is followed by one user record with its tool results.
//
// Subagents' own steps stay out, like Claude's sidechains; the spawn call
// and the subagent's answer stay in.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

type message struct {
	ID              string            `json:"id"` // user-<ms>, ai-<ms>-<hex>, sys-<ms>, …
	Variant         string            `json:"variant"`
	Content         string            `json:"content"`
	Blocks          []block           `json:"blocks"`
	Attachments     []json.RawMessage `json:"attachments"`     // images, by path
	TextAttachments []json.RawMessage `json:"textAttachments"` // pasted text, as a preview
	FileAttachments []json.RawMessage `json:"fileAttachments"` // files, by path
}

type block struct {
	Type       string          `json:"type"` // text, tool, ask-user, agent, mode-divider
	Content    string          `json:"content"`
	TextType   string          `json:"textType"` // text: "text" or "reasoning"
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Input      json.RawMessage `json:"input"`
	Output     *string         `json:"output"` // tool: nil until it finished
	Questions  json.RawMessage `json:"questions"`
	Answers    json.RawMessage `json:"answers"`
	Skipped    bool            `json:"skipped"`
	AgentID    string          `json:"agentId"`
	AgentType  string          `json:"agentType"`
	Prompt     string          `json:"prompt"`
	Params     json.RawMessage `json:"params"`
	Blocks     []block         `json:"blocks"` // agent: the subagent's own steps
}

// writeMessages writes the chat in dir, which List showed as s.
func writeMessages(dir string, s session.Session, w *portable.Writer) error {
	summary, summaryAt, err := lastSummary(filepath.Join(dir, "run-state.json"))
	if err != nil {
		return err
	}
	// Without a time the summary can't be put where it happened. Leave it
	// out: chat-messages.json still has every message it summarizes.
	if summary != "" && summaryAt.IsZero() {
		w.Drop("compaction summary without a time")
		summary = ""
	}
	f, err := os.Open(filepath.Join(dir, "chat-messages.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := expect(dec, '['); err != nil {
		return err
	}

	h := portable.Header{Agent: "freebuff", ID: s.ID, Title: s.Title, Dir: s.Dir, Updated: s.LastUsed.UTC()}
	headed := false
	c := converter{w: w}
	var at time.Time // the latest message's time, for an id without one
	for dec.More() {
		var m message
		err := dec.Decode(&m)
		if typeErr := (*json.UnmarshalTypeError)(nil); errors.As(err, &typeErr) {
			w.Drop("unreadable line") // the decoder skipped the whole value
			continue
		}
		if err != nil {
			return err
		}
		if t, ok := idTime(m.ID); ok {
			at = t
		}
		if !headed {
			h.Created, headed = at, true
			if at.IsZero() {
				h.Created = h.Updated
			}
			if err := w.Header(h); err != nil {
				return err
			}
		}
		if summary != "" && at.After(summaryAt) {
			c.write(portable.Record{Kind: "compaction", Time: summaryAt, Text: summary})
			summary = ""
		}
		c.message(m, at)
	}
	// More stops at the end of the file too, so a file cut off between two
	// messages only shows here.
	if err := expect(dec, ']'); err != nil {
		return err
	}
	if !headed {
		h.Created = h.Updated
		if err := w.Header(h); err != nil {
			return err
		}
	}
	if summary != "" {
		c.write(portable.Record{Kind: "compaction", Time: summaryAt, Text: summary})
	}
	return c.err
}

// idTime reads the milliseconds Freebuff puts in a message id.
func idTime(id string) (time.Time, bool) {
	_, rest, _ := strings.Cut(id, "-")
	for part := range strings.SplitSeq(rest, "-") {
		if len(part) != 13 {
			continue
		}
		if ms, err := strconv.ParseInt(part, 10, 64); err == nil {
			return time.UnixMilli(ms).UTC(), true
		}
	}
	return time.Time{}, false
}

// converter turns messages into records, holding one reply and its tool
// results until the reply ends.
type converter struct {
	w       *portable.Writer
	err     error
	reply   *portable.Record
	called  bool // the reply has a tool call, so the next text is a new reply
	results []portable.Part
}

func (c *converter) write(r portable.Record) {
	if c.err == nil {
		c.err = c.w.Record(r)
	}
}

func (c *converter) flush() {
	if c.reply != nil {
		c.write(*c.reply)
		if len(c.results) > 0 {
			c.write(portable.Record{Kind: "message", Role: "user", Time: c.reply.Time, Parts: c.results})
		}
	}
	c.reply, c.called, c.results = nil, false, nil
}

func (c *converter) add(p portable.Part, t time.Time) {
	if c.reply == nil {
		c.reply = &portable.Record{Kind: "message", Role: "assistant", Time: t}
	}
	c.reply.Parts = append(c.reply.Parts, p)
}

func (c *converter) message(m message, t time.Time) {
	switch {
	case m.Variant == "user":
		for range len(m.Attachments) + len(m.TextAttachments) + len(m.FileAttachments) {
			c.w.Drop("attached file")
		}
		text := m.Content
		if text == "" {
			text = words(m.Blocks)
		}
		if strings.TrimSpace(text) == "" {
			c.w.Drop("bookkeeping line")
			return
		}
		c.write(portable.Record{Kind: "message", Role: "user", Time: t, Parts: []portable.Part{{Type: "text", Text: text}}})
	case m.Variant == "ai" && strings.HasPrefix(m.ID, "sys-"):
		c.w.Drop("bookkeeping line") // Freebuff's own notices, like "Context compacted"
	case m.Variant == "ai":
		if len(m.Blocks) == 0 && m.Content != "" {
			m.Blocks = []block{{Type: "text", Content: m.Content}}
		}
		for _, b := range m.Blocks {
			c.block(b, t)
		}
		c.flush()
	case m.Variant == "agent":
		c.w.Drop("sidechain line")
	default: // errors, and anything newer
		c.w.Drop("bookkeeping line")
	}
}

func (c *converter) block(b block, t time.Time) {
	switch b.Type {
	case "text":
		if b.Content == "" {
			return
		}
		if c.called {
			c.flush()
		}
		kind := "text"
		if b.TextType == "reasoning" {
			kind = "reasoning"
		}
		c.add(portable.Part{Type: kind, Text: b.Content}, t)
	case "tool":
		c.call(portable.Part{Type: "tool_call", ID: b.ToolCallID, Name: b.ToolName, Input: b.Input}, t)
		if b.Output != nil { // a tool still running when the chat was saved has none
			c.results = append(c.results, portable.Part{Type: "tool_result", ID: b.ToolCallID, Text: *b.Output})
		}
	case "ask-user": // Freebuff's ask_user tool, once answered
		input, _ := json.Marshal(struct {
			Questions json.RawMessage `json:"questions,omitempty"`
		}{b.Questions})
		c.call(portable.Part{Type: "tool_call", ID: b.ToolCallID, Name: "ask_user", Input: input}, t)
		output, _ := json.Marshal(struct {
			Answers json.RawMessage `json:"answers,omitempty"`
			Skipped bool            `json:"skipped,omitempty"`
		}{b.Answers, b.Skipped})
		c.results = append(c.results, portable.Part{Type: "tool_result", ID: b.ToolCallID, Text: string(output)})
	case "agent": // one subagent from spawn_agents, which has no tool block of its own
		type spawn struct {
			AgentType string          `json:"agent_type"`
			Prompt    string          `json:"prompt,omitempty"`
			Params    json.RawMessage `json:"params,omitempty"`
		}
		input, _ := json.Marshal(map[string][]spawn{"agents": {{b.AgentType, b.Prompt, b.Params}}})
		c.call(portable.Part{Type: "tool_call", ID: b.AgentID, Name: "spawn_agents", Input: input}, t)
		for _, step := range b.Blocks {
			if step.Type != "text" || step.TextType == "reasoning" {
				c.w.Drop("sidechain line")
			}
		}
		if answer := words(b.Blocks); answer != "" {
			c.results = append(c.results, portable.Part{Type: "tool_result", ID: b.AgentID, Text: answer})
		}
	default: // mode dividers, and anything newer
		c.w.Drop("bookkeeping line")
	}
}

func (c *converter) call(p portable.Part, t time.Time) {
	c.add(p, t)
	c.called = true
}

// words joins the text blocks that aren't reasoning.
func words(blocks []block) string {
	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && b.TextType != "reasoning" && b.Content != "" {
			texts = append(texts, b.Content)
		}
	}
	return strings.Join(texts, "\n\n")
}

// lastSummary finds the compaction summary in run-state.json, the user
// message tagged CONVERSATION_SUMMARY that starts the model's history after
// a compaction, and when it was made: zero when it has no sentAt.
// A chat never compacted has none.
func lastSummary(path string) (string, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", time.Time{}, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = walk(dec, "sessionState", "mainAgentState", "messageHistory")
	if errors.Is(err, errNoKey) {
		return "", time.Time{}, nil
	}
	if err != nil {
		return "", time.Time{}, err
	}
	for dec.More() {
		var m struct {
			Role    string          `json:"role"`
			Tags    []string        `json:"tags"` // the summary's is CONVERSATION_SUMMARY
			SentAt  float64         `json:"sentAt"`
			Content json.RawMessage `json:"content"` // text, or a list of parts
		}
		if err := dec.Decode(&m); err != nil {
			return "", time.Time{}, err
		}
		// A prompt can quote the summary markup; only Freebuff's own summary
		// carries its tag.
		if m.Role != "user" || !slices.Contains(m.Tags, "CONVERSATION_SUMMARY") {
			continue
		}
		if text, ok := summaryText(m.Content); ok {
			if m.SentAt == 0 {
				return text, time.Time{}, nil
			}
			return text, time.UnixMilli(int64(m.SentAt)).UTC(), nil
		}
	}
	// More stops at the end of the file too, so a cut off file only shows here.
	if err := expect(dec, ']'); err != nil {
		return "", time.Time{}, err
	}
	return "", time.Time{}, nil
}

// summaryText reads a message Freebuff wrote as
// <conversation_summary> … <historical_memory>text</historical_memory> …
func summaryText(content json.RawMessage) (string, bool) {
	var text string
	if json.Unmarshal(content, &text) != nil {
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		json.Unmarshal(content, &parts)
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				b.WriteString(p.Text)
			}
		}
		text = b.String()
	}
	_, text, ok := strings.Cut(text, "<conversation_summary>")
	if !ok {
		return "", false
	}
	text, _, _ = strings.Cut(text, "</conversation_summary>")
	if _, memory, ok := strings.Cut(text, "<historical_memory>"); ok {
		text, _, _ = strings.Cut(memory, "</historical_memory>")
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}
