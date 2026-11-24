// Package codex reads OpenAI Codex CLI sessions.
package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads sessions from Home, usually ~/.codex.
type Source struct {
	Home string
}

func (Source) Name() string { return "codex" }

func (c Source) List() ([]session.Session, error) {
	dir := filepath.Join(c.Home, "sessions")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, session.ErrMissing
	}

	names := threadNames(filepath.Join(c.Home, "session_index.jsonl"))

	var list []session.Session
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasPrefix(d.Name(), "rollout-") || !strings.HasSuffix(d.Name(), ".jsonl") {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		r, err := parseFile(path)
		if err != nil || r.ID == "" || r.Subagent {
			return nil
		}

		title := names[r.ID]
		if title == "" {
			title = helpers.Truncate(r.Prompt, 60)
		}
		list = append(list, session.Session{
			Agent:    "codex",
			ID:       r.ID,
			Title:    title,
			Dir:      r.Cwd,
			LastUsed: info.ModTime(),
		})
		return nil
	})
	return list, err
}

func (Source) ResumeCommand(s session.Session) []string {
	return []string{"codex", "resume", s.ID}
}

func (Source) Delete(s session.Session, trashDir string) error {
	return errors.New("delete not supported for codex yet")
}

type rollout struct {
	ID, Cwd, Prompt string
	Subagent        bool
}

type line struct {
	Type    string `json:"type"`
	Payload struct {
		ID      string          `json:"id"`
		Cwd     string          `json:"cwd"`
		Source  json.RawMessage `json:"source"`
		Type    string          `json:"type"`
		Message string          `json:"message"`
	} `json:"payload"`
}

var (
	metaMark   = []byte(`"session_meta"`)
	promptMark = []byte(`"user_message"`)
)

// parseFile reads the session meta and the first user prompt, then stops.
func parseFile(path string) (rollout, error) {
	var r rollout
	f, err := os.Open(path)
	if err != nil {
		return r, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	for r.ID == "" || r.Prompt == "" {
		raw, readErr := br.ReadBytes('\n')

		var l line
		switch {
		case r.ID == "" && bytes.Contains(raw, metaMark):
			if json.Unmarshal(raw, &l) == nil && l.Type == "session_meta" {
				r.ID = l.Payload.ID
				r.Cwd = l.Payload.Cwd
				r.Subagent = bytes.Contains(l.Payload.Source, []byte(`"subagent"`))
			}
		case r.Prompt == "" && bytes.Contains(raw, promptMark):
			if json.Unmarshal(raw, &l) == nil && l.Payload.Type == "user_message" {
				r.Prompt = l.Payload.Message
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return r, readErr
		}
	}
	return r, nil
}

// threadNames maps session id to its name; later lines win.
func threadNames(path string) map[string]string {
	names := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return names
	}
	for raw := range bytes.SplitSeq(data, []byte("\n")) {
		var v struct {
			ID         string `json:"id"`
			ThreadName string `json:"thread_name"`
		}
		if json.Unmarshal(raw, &v) == nil && v.ID != "" && v.ThreadName != "" {
			names[v.ID] = v.ThreadName
		}
	}
	return names
}
