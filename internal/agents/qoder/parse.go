// Package qoder reads Qoder CLI sessions.
package qoder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
)

type line struct {
	Type    string `json:"type"`
	AITitle string `json:"aiTitle"`
	Cwd     string `json:"cwd"`
	IsMeta  bool   `json:"isMeta"`
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

var (
	titleMark = []byte(`"type":"ai-title"`)
	userMark  = []byte(`"type":"user"`)
)

// parseFile returns the session's title and folder.
// Title order: first ai-title, then first prompt.
func parseFile(path string) (title, cwd string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	var firstPrompt string
	r := bufio.NewReader(f)
	for title == "" || cwd == "" {
		raw, readErr := r.ReadBytes('\n')

		var l line
		switch {
		case title == "" && bytes.Contains(raw, titleMark):
			if json.Unmarshal(raw, &l) == nil {
				title = l.AITitle
			}
		case (cwd == "" || firstPrompt == "") && bytes.Contains(raw, userMark):
			if json.Unmarshal(raw, &l) == nil {
				if cwd == "" {
					cwd = l.Cwd
				}
				if firstPrompt == "" && !l.IsMeta {
					firstPrompt = prompt(text(l.Message.Content))
				}
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", "", readErr
		}
	}

	if title == "" {
		title = helpers.Truncate(firstPrompt, 60, "…")
	}
	return title, cwd, nil
}

// prompt turns a slash command into "/name" and skips other tagged lines.
func prompt(s string) string {
	if name := helpers.Between(s, "<command-name>", "</command-name>"); name != "" {
		return name
	}
	if strings.HasPrefix(s, "<") {
		return ""
	}
	return s
}

// text gets the words from content, which is a string or a list of parts.
func text(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &parts) == nil {
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				return p.Text
			}
		}
	}
	return ""
}
