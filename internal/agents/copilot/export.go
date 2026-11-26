package copilot

// Export reads a Copilot CLI session's event log. Copilot has no export
// command, and session-store.db keeps only each turn's words, so the
// conversation comes from session-state/<id>/events.jsonl.
//
// The log is a list of events. Subagents write into the same log, marked
// with parentToolCallId (or parentAgentTaskId on their prompt); they stay
// out, like Claude's sidechains, while the parent's tool call and result
// stay in.

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

type event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		Content           string                    `json:"content"`
		Model             string                    `json:"model"`
		ReasoningText     string                    `json:"reasoningText"`
		ReasoningOpaque   string                    `json:"reasoningOpaque"`
		ToolRequests      []toolRequest             `json:"toolRequests"`
		ToolCallID        string                    `json:"toolCallId"`
		Success           bool                      `json:"success"`
		Result            *struct{ Content string } `json:"result"`
		Error             json.RawMessage           `json:"error"` // {message} on tools, plain text on subagent.failed
		Attachments       []json.RawMessage         `json:"attachments"`
		ParentToolCallID  string                    `json:"parentToolCallId"`
		ParentAgentTaskID string                    `json:"parentAgentTaskId"`
	} `json:"data"`
}

type toolRequest struct {
	ToolCallID string          `json:"toolCallId"`
	Name       string          `json:"name"`
	Arguments  json.RawMessage `json:"arguments"`
}

// header reads the session's title and folder from session-store.db, the
// row List shows.
func (c Source) header(id string) (portable.Header, error) {
	db, err := sql.Open("sqlite", "file:"+c.DBPath+"?mode=ro")
	if err != nil {
		return portable.Header{}, err
	}
	defer db.Close()
	var cwd string
	var summary sql.NullString
	err = db.QueryRow(`SELECT cwd, summary FROM sessions WHERE id = ?`, id).Scan(&cwd, &summary)
	if errors.Is(err, sql.ErrNoRows) {
		return portable.Header{}, fmt.Errorf("copilot session %q not found", id)
	}
	if err != nil {
		return portable.Header{}, err
	}
	title := summary.String
	if title == "" {
		title = "(untitled)"
	}
	return portable.Header{Agent: "copilot", ID: id, Title: title, Dir: cwd}, nil
}

// eventsFile is the session's event log. The id is part of a path, so one
// that could leave session-state is refused.
func (c Source) eventsFile(id string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("invalid copilot session id %q", id)
	}
	return filepath.Join(filepath.Dir(c.DBPath), "session-state", id, "events.jsonl"), nil
}

// scanEvents calls each for every line; ok is false for a line that isn't
// JSON. A missing log is an empty session: Copilot hadn't written one yet.
func scanEvents(path string, each func(e event, ok bool)) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	for sc.Scan() {
		var e event
		ok := json.Unmarshal(sc.Bytes(), &e) == nil
		each(e, ok)
	}
	return sc.Err()
}

// times is the first and last event time, for the header.
func times(path string) (first, last time.Time, err error) {
	err = scanEvents(path, func(e event, ok bool) {
		if !ok || e.Timestamp.IsZero() {
			return
		}
		if first.IsZero() {
			first = e.Timestamp
		}
		last = e.Timestamp
	})
	return first, last, err
}

// writeEvents writes the log's conversation as records: each assistant
// message is one reply, followed by one user record with its tool results.
func writeEvents(path string, w *portable.Writer) error {
	var reply *portable.Record
	var results []portable.Part
	var writeErr error
	write := func(r portable.Record) {
		if writeErr == nil {
			writeErr = w.Record(r)
		}
	}
	flush := func() {
		if reply != nil {
			write(*reply)
		}
		if len(results) > 0 {
			write(portable.Record{Kind: "message", Role: "user", Time: reply.Time, Parts: results})
		}
		reply, results = nil, nil
	}

	err := scanEvents(path, func(e event, ok bool) {
		d := e.Data
		switch {
		case !ok:
			w.Drop("unreadable line")
		case d.ParentToolCallID != "" || d.ParentAgentTaskID != "":
			w.Drop("sidechain line")
		case e.Type == "system.message":
			w.Drop("injected text")
		case e.Type == "user.message":
			for range d.Attachments { // path references, not file contents
				w.Drop("attached file")
			}
			if strings.TrimSpace(d.Content) == "" {
				return
			}
			flush()
			write(portable.Record{Kind: "message", Role: "user", Time: e.Timestamp, Parts: []portable.Part{{Type: "text", Text: d.Content}}})
		case e.Type == "assistant.message":
			flush()
			r := portable.Record{Kind: "message", Role: "assistant", Time: e.Timestamp, Model: d.Model}
			switch {
			case d.ReasoningText != "":
				r.Parts = append(r.Parts, portable.Part{Type: "reasoning", Text: d.ReasoningText})
			case d.ReasoningOpaque != "":
				w.Drop("thinking without text")
			}
			if d.Content != "" {
				r.Parts = append(r.Parts, portable.Part{Type: "text", Text: d.Content})
			}
			for _, t := range d.ToolRequests {
				r.Parts = append(r.Parts, portable.Part{Type: "tool_call", ID: t.ToolCallID, Name: t.Name, Input: t.Arguments})
			}
			if len(r.Parts) > 0 {
				reply = &r
			}
		case e.Type == "tool.execution_complete":
			res := portable.Part{Type: "tool_result", ID: d.ToolCallID, Error: !d.Success}
			if d.Result != nil {
				res.Text = d.Result.Content
			} else {
				res.Text = errorText(d.Error)
			}
			if reply == nil {
				write(portable.Record{Kind: "message", Role: "user", Time: e.Timestamp, Parts: []portable.Part{res}})
				return
			}
			results = append(results, res)
		default: // turn markers, tool starts, permissions, session events
			w.Drop("bookkeeping line")
		}
	})
	if err != nil {
		return err
	}
	flush()
	return writeErr
}

// errorText reads an error written as {message} or as plain text.
func errorText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Message string `json:"message"`
	}
	json.Unmarshal(raw, &obj)
	return obj.Message
}
