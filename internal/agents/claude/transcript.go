package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

type record struct {
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	IsSidechain bool      `json:"isSidechain"`
	IsMeta      bool      `json:"isMeta"`
	// A summary written when the conversation was compacted, not a turn.
	IsCompactSummary bool `json:"isCompactSummary"`
	Message          struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type part struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (c Source) Transcript(s session.Session) (session.Transcript, error) {
	t := session.Transcript{Session: s}
	path, err := findFile(c.Home, s.ID)
	if err != nil {
		return t, err
	}
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		raw, readErr := r.ReadBytes('\n')
		if raw = bytes.TrimSpace(raw); len(raw) > 0 {
			var rec record
			if json.Unmarshal(raw, &rec) != nil {
				t.Skipped++
			} else {
				for _, m := range messages(rec) {
					t.Add(m)
				}
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

func (Source) StartCommand(dir, prompt string) []string {
	return []string{"claude", prompt}
}

func findFile(home, id string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(home, "projects", "*", id+".jsonl"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("claude session %s not found under %s", id, filepath.Join(home, "projects"))
	}
	return matches[0], nil
}

// messages keeps prose and tool calls of the main conversation only, one per
// part in order: thinking, tool results, subagent and meta records are dropped.
func messages(rec record) []session.Message {
	if rec.IsSidechain || rec.IsMeta || rec.IsCompactSummary || (rec.Type != "user" && rec.Type != "assistant") {
		return nil
	}
	at := rec.Timestamp

	var s string
	if json.Unmarshal(rec.Message.Content, &s) == nil {
		return []session.Message{{Role: rec.Type, Time: at, Text: userText(rec.Type, s)}}
	}
	var parts []part
	if json.Unmarshal(rec.Message.Content, &parts) != nil {
		return nil
	}
	var out []session.Message
	for _, p := range parts {
		switch p.Type {
		case "text":
			out = append(out, session.Message{Role: rec.Type, Time: at, Text: userText(rec.Type, p.Text)})
		case "tool_use":
			out = append(out, session.Message{Role: rec.Type, Time: at, Tools: []string{transcript.ToolLine(p.Name, p.Input)}})
		}
	}
	return out
}

// injected are the tags the agent wraps around text it adds to the person's
// turn (command output, task notices, editor context), not what they typed.
var injected = []string{
	"task-notification", "bash-stdout", "bash-stderr", "system-reminder",
	"local-command-stdout", "local-command-stderr", "local-command-caveat",
	"ide_opened_file", "ide_selection", "ide_diagnostics",
}

// userText turns a slash command into "/name args" and a shell escape into
// "! command", and drops injected text.
func userText(role, s string) string {
	s = strings.TrimSpace(s)
	if role != "user" {
		return s
	}
	if name := helpers.Between(s, "<command-name>", "</command-name>"); name != "" {
		return strings.TrimSpace(name + " " + helpers.Between(s, "<command-args>", "</command-args>"))
	}
	if cmd := helpers.Between(s, "<bash-input>", "</bash-input>"); cmd != "" {
		return "! " + cmd
	}
	// The agent records an interrupted reply as a user turn of its own.
	if strings.HasPrefix(s, "[Request interrupted by user") {
		return ""
	}
	return transcript.StripInjected(s, injected)
}
