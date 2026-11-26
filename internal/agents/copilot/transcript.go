package copilot

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

type event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		Content          string          `json:"content"`
		ToolName         string          `json:"toolName"`
		Arguments        json.RawMessage `json:"arguments"`
		ParentToolCallID string          `json:"parentToolCallId"`
	} `json:"data"`
}

// Transcript reads session-state/<id>/events.jsonl beside the database,
// and falls back to the turns table when that file is gone or holds no messages.
func (c Source) Transcript(s session.Session) (session.Transcript, error) {
	t, err := c.readEvents(s)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(t.Messages) == 0) {
		t.Messages = nil
		return t, c.readTurns(&t)
	}
	return t, err
}

func (c Source) readEvents(s session.Session) (session.Transcript, error) {
	t := session.Transcript{Session: s}
	f, err := os.Open(filepath.Join(filepath.Dir(c.DBPath), "session-state", s.ID, "events.jsonl"))
	if err != nil {
		return t, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		raw, readErr := r.ReadBytes('\n')
		if raw = bytes.TrimSpace(raw); len(raw) > 0 {
			var e event
			if json.Unmarshal(raw, &e) != nil {
				t.Skipped++
			} else if e.Data.ParentToolCallID == "" {
				// Events under a parent tool call come from subagents.
				addEvent(&t, e)
			}
		}
		if errors.Is(readErr, io.EOF) {
			return t, nil
		}
		if readErr != nil {
			return t, readErr
		}
	}
}

func addEvent(t *session.Transcript, e event) {
	switch e.Type {
	case "user.message":
		t.Add(session.Message{Role: "user", Time: e.Timestamp, Text: e.Data.Content})
	case "assistant.message":
		t.Add(session.Message{Role: "assistant", Time: e.Timestamp, Text: e.Data.Content})
	case "tool.execution_start":
		t.Add(session.Message{Role: "assistant", Time: e.Timestamp, Tools: []string{transcript.ToolLine(e.Data.ToolName, e.Data.Arguments)}})
	}
}

func (c Source) readTurns(t *session.Transcript) error {
	db, err := sql.Open("sqlite", "file:"+c.DBPath+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT user_message, assistant_response, timestamp FROM turns WHERE session_id = ? ORDER BY turn_index`, t.Session.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var user, reply, stamp sql.NullString
		if err := rows.Scan(&user, &reply, &stamp); err != nil {
			return err
		}
		at, _ := time.Parse(time.RFC3339Nano, stamp.String)
		t.Add(session.Message{Role: "user", Time: at, Text: user.String})
		t.Add(session.Message{Role: "assistant", Text: reply.String})
	}
	return rows.Err()
}

func (Source) StartCommand(dir, prompt string) []string {
	return []string{"copilot", "-i", prompt}
}
