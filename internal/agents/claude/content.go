package claude

// Export: turn a message's content into portable parts.
//
// Content is a string or a list of blocks. User turns also carry blocks
// Claude Code injected (reminders, local command output), which are removed;
// assistant text is kept word for word.

import (
	"encoding/json"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

// contentBlock is one block of a message's content.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
	Source    struct {
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source"`
}

func blocks(content json.RawMessage) []contentBlock {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return []contentBlock{{Type: "text", Text: s}}
	}
	var list []contentBlock
	json.Unmarshal(content, &list)
	return list
}

func assistantParts(content json.RawMessage, w *portable.Writer) []portable.Part {
	var parts []portable.Part
	for _, b := range blocks(content) {
		switch b.Type {
		case "text":
			parts = append(parts, portable.Part{Type: "text", Text: b.Text})
		case "thinking":
			// Claude often keeps only the signature; there's nothing to carry.
			if b.Thinking == "" {
				w.Drop("thinking without text")
				continue
			}
			parts = append(parts, portable.Part{Type: "reasoning", Text: b.Thinking})
		case "tool_use":
			parts = append(parts, portable.Part{Type: "tool_call", ID: b.ID, Name: b.Name, Input: b.Input})
		case "image":
			parts = append(parts, image(b, ""))
		}
	}
	return parts
}

// userParts splits a user turn into tool results (with their images) and
// what the user said.
func userParts(content json.RawMessage, w *portable.Writer) (results, said []portable.Part) {
	for _, b := range blocks(content) {
		switch b.Type {
		case "text":
			if t := userText(b.Text, w); t != "" {
				said = append(said, portable.Part{Type: "text", Text: t})
			}
		case "image":
			said = append(said, image(b, ""))
		case "tool_result":
			results = append(results, toolResult(b)...)
		}
	}
	return results, said
}

// toolResult is the result part, then one image part per image in it with
// the result's id, so an importer can put them back inside the result.
func toolResult(b contentBlock) []portable.Part {
	res := portable.Part{Type: "tool_result", ID: b.ToolUseID, Error: b.IsError}
	var images []portable.Part
	var text strings.Builder
	for _, x := range blocks(b.Content) {
		switch x.Type {
		case "text":
			text.WriteString(x.Text)
		case "image":
			images = append(images, image(x, b.ToolUseID))
		}
	}
	res.Text = text.String()
	return append([]portable.Part{res}, images...)
}

func image(b contentBlock, id string) portable.Part {
	return portable.Part{Type: "image", ID: id, MediaType: b.Source.MediaType, Data: b.Source.Data}
}

// injectedTags are blocks Claude Code adds to a user turn; the user never
// typed them.
var injectedTags = []string{"system-reminder", "local-command-caveat", "local-command-stdout", "local-command-stderr"}

// userText removes injected blocks and turns a slash command back into what
// the user typed.
func userText(s string, w *portable.Writer) string {
	if name := helpers.Between(s, "<command-name>", "</command-name>"); name != "" {
		args := helpers.Between(s, "<command-args>", "</command-args>")
		return strings.TrimSpace(name + " " + args)
	}
	for _, tag := range injectedTags {
		open, close := "<"+tag+">", "</"+tag+">"
		for {
			start := strings.Index(s, open)
			if start < 0 {
				break
			}
			end := strings.Index(s[start:], close)
			if end < 0 {
				s = s[:start] // unterminated: the rest is the block
			} else {
				s = s[:start] + s[start+end+len(close):]
			}
			w.Drop("injected text")
		}
	}
	return strings.TrimSpace(s)
}
