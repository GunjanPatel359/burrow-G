package opencode

// Export uses OpenCode's own `opencode session export`, which prints the
// session as structured JSON. Reading its database instead would mean
// following every migration; the export command stays working because
// OpenCode's own import depends on it.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

// Exec runs a command and returns its output, with stderr in the error. It
// is what main.go passes as Source.Run.
//
// Output goes to a temp file, not a pipe: OpenCode exits before a pipe is
// drained and silently cuts large exports short (a 17 MB session arrived as
// 16 MiB + 64 KiB, exit status 0). Written to a file, it arrives whole.
func Exec(name string, args ...string) ([]byte, error) {
	f, err := os.CreateTemp("", "burrow-g-*.out") // mode 600: it holds a session
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	var stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = f, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return os.ReadFile(f.Name())
}

// exported is the JSON `opencode session export` prints.
type exported struct {
	Info struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Location struct {
			Directory string `json:"directory"`
		} `json:"location"`
		Time stamps `json:"time"`
	} `json:"info"`
	Messages []struct {
		Type    string              `json:"type"`
		Time    stamps              `json:"time"`
		Text    string              `json:"text"`    // user, system, synthetic
		Summary string              `json:"summary"` // compaction
		Files   []file              `json:"files"`   // user
		Model   struct{ ID string } `json:"model"`
		Content []part              `json:"content"` // assistant
	} `json:"messages"`
}

type stamps struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

type file struct {
	Data string `json:"data"`
	Mime string `json:"mime"`
}

type part struct {
	Type  string `json:"type"` // text, reasoning, tool
	Text  string `json:"text"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	State struct {
		Status  string          `json:"status"` // completed, error, or not finished
		Input   json.RawMessage `json:"input"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"state"`
}

func ms(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return time.UnixMilli(v).UTC()
}

// export runs OpenCode's export for id and writes it in the portable format.
func export(run func(string, ...string) ([]byte, error), id string, w *portable.Writer) error {
	out, err := run("opencode", "session", "export", "--standalone", id)
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("opencode not found on PATH")
	}
	if err != nil {
		return fmt.Errorf("opencode session export: %w", err)
	}
	var doc exported
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&doc); err != nil {
		return fmt.Errorf("opencode session export: unexpected output: %w", err)
	}

	if err := w.Header(portable.Header{
		Agent: "opencode", ID: id, Title: doc.Info.Title, Dir: doc.Info.Location.Directory,
		Created: ms(doc.Info.Time.Created), Updated: ms(doc.Info.Time.Updated),
	}); err != nil {
		return err
	}
	for _, m := range doc.Messages {
		at := ms(m.Time.Created)
		var recs []portable.Record
		switch m.Type {
		case "user":
			recs = []portable.Record{userRecord(m.Text, m.Files, at, w)}
		case "assistant":
			recs = assistantRecords(m.Content, m.Model.ID, at)
		case "compaction":
			recs = []portable.Record{{Kind: "compaction", Time: at, Text: m.Summary}}
		case "system", "synthetic": // OpenCode wrote these, not the user
			w.Drop("injected text")
		default: // idle, model-switched, agent-switched, …
			w.Drop("bookkeeping line")
		}
		for _, r := range recs {
			if err := w.Record(r); err != nil {
				return err
			}
		}
	}
	return nil
}

func userRecord(text string, files []file, at time.Time, w *portable.Writer) portable.Record {
	r := portable.Record{Kind: "message", Role: "user", Time: at}
	if text != "" {
		r.Parts = append(r.Parts, portable.Part{Type: "text", Text: text})
	}
	for _, f := range files {
		if !strings.HasPrefix(f.Mime, "image/") {
			w.Drop("attached file")
			continue
		}
		r.Parts = append(r.Parts, portable.Part{Type: "image", MediaType: f.Mime, Data: f.Data})
	}
	return r
}

// assistantRecords is the reply, then one user record with its finished
// tool results in call order: the same grouping as every agent's export.
func assistantRecords(content []part, model string, at time.Time) []portable.Record {
	reply := portable.Record{Kind: "message", Role: "assistant", Time: at, Model: model}
	var results []portable.Part
	for _, p := range content {
		switch p.Type {
		case "text", "reasoning":
			reply.Parts = append(reply.Parts, portable.Part{Type: p.Type, Text: p.Text})
		case "tool":
			reply.Parts = append(reply.Parts, portable.Part{Type: "tool_call", ID: p.ID, Name: p.Name, Input: p.State.Input})
			switch p.State.Status {
			case "completed":
				var text strings.Builder
				for _, c := range p.State.Content {
					text.WriteString(c.Text)
				}
				results = append(results, portable.Part{Type: "tool_result", ID: p.ID, Text: text.String()})
			case "error":
				results = append(results, portable.Part{Type: "tool_result", ID: p.ID, Text: p.State.Error.Message, Error: true})
			}
		}
	}
	recs := []portable.Record{reply}
	if len(results) > 0 {
		recs = append(recs, portable.Record{Kind: "message", Role: "user", Time: at, Parts: results})
	}
	return recs
}
