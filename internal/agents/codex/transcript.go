package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

type item struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Payload   struct {
		Type      string          `json:"type"`
		Role      string          `json:"role"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Input     json.RawMessage `json:"input"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"payload"`
}

// injected marks user text that Codex adds itself, not the person.
var injected = []string{"<", "# AGENTS.md instructions"}

func (c Source) Transcript(s session.Session) (session.Transcript, error) {
	t := session.Transcript{Session: s}
	path, err := c.findRollout(s.ID)
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
			var it item
			if json.Unmarshal(raw, &it) != nil {
				t.Skipped++
			} else if it.Type == "response_item" {
				addItem(&t, it)
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

func addItem(t *session.Transcript, it item) {
	p := it.Payload
	switch p.Type {
	case "message":
		if p.Role != "user" && p.Role != "assistant" {
			return // developer and system prompts
		}
		var texts []string
		for _, c := range p.Content {
			if (c.Type == "input_text" || c.Type == "output_text") && !isInjected(c.Text) {
				texts = append(texts, c.Text)
			}
		}
		t.Add(session.Message{Role: p.Role, Time: it.Timestamp, Text: strings.Join(texts, "\n\n")})
	case "function_call":
		t.Add(session.Message{Role: "assistant", Time: it.Timestamp, Tools: []string{transcript.ToolLine(p.Name, p.Arguments)}})
	case "custom_tool_call":
		t.Add(session.Message{Role: "assistant", Time: it.Timestamp, Tools: []string{transcript.ToolLine(p.Name, customInput(p.Input))}})
	}
}

var (
	execCmd  = regexp.MustCompile(`cmd:\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')`)
	execTool = regexp.MustCompile(`tools\.(\w+)\(`)
)

// customInput turns a custom tool's input into JSON ToolLine can read. The
// exec tool takes JavaScript such as tools.exec_command({cmd:"go test"}), so
// its first shell command becomes the argument, else the inner tool's name.
func customInput(raw json.RawMessage) json.RawMessage {
	var src string
	if json.Unmarshal(raw, &src) != nil || json.Valid([]byte(src)) {
		return raw
	}
	arg := ""
	if cmds := execCmd.FindAllStringSubmatch(src, -1); len(cmds) > 0 {
		arg = unquote(cmds[0][1])
		if len(cmds) > 1 {
			arg += fmt.Sprintf(" (+%d more)", len(cmds)-1)
		}
	} else if m := execTool.FindStringSubmatch(src); m != nil {
		arg = m[1]
	}
	if arg == "" {
		return raw
	}
	out, _ := json.Marshal(map[string]string{"cmd": arg})
	return out
}

// unquote reads a JavaScript string literal, keeping it as written if it
// uses an escape Go doesn't know.
func unquote(lit string) string {
	if lit[0] == '\'' {
		lit = `"` + strings.ReplaceAll(strings.ReplaceAll(lit[1:len(lit)-1], `\'`, `'`), `"`, `\"`) + `"`
	}
	if s, err := strconv.Unquote(lit); err == nil {
		return s
	}
	return lit[1 : len(lit)-1]
}

func isInjected(s string) bool {
	s = strings.TrimSpace(s)
	for _, prefix := range injected {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// findRollout finds sessions/**/rollout-<time>-<id>.jsonl.
func (c Source) findRollout(id string) (string, error) {
	var found string
	suffix := "-" + id + ".jsonl"
	err := filepath.WalkDir(filepath.Join(c.Home, "sessions"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), suffix) {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("codex session %s not found under %s", id, filepath.Join(c.Home, "sessions"))
	}
	return found, nil
}

func (Source) StartCommand(dir, prompt string) []string {
	return []string{"codex", prompt}
}
