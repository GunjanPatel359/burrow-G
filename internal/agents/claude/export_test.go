package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// exportFile puts fixture in a Claude home as session id, exports it and
// reads the result back.
func exportFile(t *testing.T, fixture, id string) (string, portable.Header, []portable.Record, portable.End) {
	t.Helper()
	home := t.TempDir()
	copyFile(t, fixture, filepath.Join(home, "projects", "-tmp-p", id+".jsonl"))

	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	if err := (Source{Home: home}).Export(session.Session{Agent: "claude", ID: id}, w); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := portable.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	var records []portable.Record
	for {
		rec, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, rec)
	}
	return home, r.Header(), records, r.End()
}

// texts lists each record's words: text parts for messages, the summary for
// a compaction (marked "compaction: ").
func texts(records []portable.Record) []string {
	var out []string
	for _, r := range records {
		if r.Kind == "compaction" {
			out = append(out, "compaction: "+r.Text)
			continue
		}
		for _, p := range r.Parts {
			if p.Type == "text" {
				out = append(out, p.Text)
			}
		}
	}
	return out
}

func TestExportSegments(t *testing.T) {
	_, _, records, end := exportFile(t, "testdata/export/segments.jsonl", "seg")

	want := []string{"first", "one", "compaction: Summary text", "second", "two"}
	if got := texts(records); !slices.Equal(got, want) {
		t.Errorf("texts = %q\nwant %q", got, want)
	}
	if end.Dropped["unreadable line"] != 1 {
		t.Errorf("unreadable line = %d, want 1", end.Dropped["unreadable line"])
	}
	if end.Dropped["sidechain line"] != 1 {
		t.Errorf("sidechain line = %d, want 1", end.Dropped["sidechain line"])
	}
	if end.Dropped["bookkeeping line"] < 1 {
		t.Errorf("bookkeeping line = %d, want at least 1", end.Dropped["bookkeeping line"])
	}
}

func TestExportHeader(t *testing.T) {
	home, h, _, _ := exportFile(t, "testdata/export/segments.jsonl", "seg")

	listed, err := Source{Home: home}.List()
	if err != nil || len(listed) != 1 {
		t.Fatalf("List() = %v, %v", listed, err)
	}
	if h.Agent != "claude" || h.ID != "seg" {
		t.Errorf("header agent/id = %q/%q", h.Agent, h.ID)
	}
	if h.Title != listed[0].Title || h.Dir != listed[0].Dir {
		t.Errorf("header title/dir = %q/%q, List shows %q/%q", h.Title, h.Dir, listed[0].Title, listed[0].Dir)
	}
	if want := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC); !h.Created.Equal(want) {
		t.Errorf("Created = %v, want %v", h.Created, want)
	}
	if want := time.Date(2026, 11, 1, 10, 0, 10, 0, time.UTC); !h.Updated.Equal(want) {
		t.Errorf("Updated = %v, want %v", h.Updated, want)
	}
}

func TestExportKeepsSideBranchResult(t *testing.T) {
	_, _, records, _ := exportFile(t, "testdata/export/parallel.jsonl", "par")

	var results []string
	for _, r := range records {
		for _, p := range r.Parts {
			if p.Type == "tool_result" {
				results = append(results, p.ID+"="+p.Text)
			}
		}
	}
	slices.Sort(results)
	if want := []string{"ta=outA", "tb=outB"}; !slices.Equal(results, want) {
		t.Errorf("tool results = %q, want %q", results, want)
	}
}

func TestExportNotFound(t *testing.T) {
	w := portable.NewWriter(io.Discard, "test")
	err := Source{Home: t.TempDir()}.Export(session.Session{ID: "nope"}, w)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("Export() = %v, want an error naming the id", err)
	}
}

func TestExportRefusesPathID(t *testing.T) {
	home := t.TempDir()
	// A real session one level up must stay out of reach of a crafted id.
	copyFile(t, "testdata/export/segments.jsonl", filepath.Join(home, "projects", "x.jsonl"))
	for _, id := range []string{"../x", "a/b", "*", "..", ""} {
		w := portable.NewWriter(io.Discard, "test")
		if err := (Source{Home: home}).Export(session.Session{ID: id}, w); err == nil {
			t.Errorf("Export(%q) succeeded, want an error", id)
		}
	}
}

// sameRecords compares records without their times.
func sameRecords(t *testing.T, got, want []portable.Record) {
	t.Helper()
	for i := range got {
		got[i].Time = time.Time{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("records:\n%s\nwant:\n%s", dump(got), dump(want))
	}
}

func dump(records []portable.Record) string {
	var b strings.Builder
	for _, r := range records {
		data, _ := json.Marshal(r)
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestExportGroupsParallelCalls(t *testing.T) {
	_, _, records, _ := exportFile(t, "testdata/export/parallel.jsonl", "par")

	sameRecords(t, records, []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "go"}}},
		{Kind: "message", Role: "assistant", Model: "claude-test", Parts: []portable.Part{
			{Type: "tool_call", ID: "ta", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`)},
			{Type: "tool_call", ID: "tb", Name: "Bash", Input: json.RawMessage(`{"command":"pwd"}`)},
		}},
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "ta", Text: "outA"},
			{Type: "tool_result", ID: "tb", Text: "outB"},
		}},
		{Kind: "message", Role: "assistant", Model: "claude-test", Parts: []portable.Part{{Type: "text", Text: "done"}}},
	})
}

func TestExportContent(t *testing.T) {
	_, _, records, end := exportFile(t, "testdata/export/content.jsonl", "con")

	sameRecords(t, records, []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "real words"}}},
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "/review 42"}}},
		{Kind: "message", Role: "assistant", Model: "claude-test", Parts: []portable.Part{
			{Type: "reasoning", Text: "plan"},
			{Type: "text", Text: "hi"},
			{Type: "tool_call", ID: "t1", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`)},
		}},
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "t1", Text: "ok"},
			{Type: "image", ID: "t1", MediaType: "image/png", Data: "iVBO"},
		}},
		// Cut off mid call: the call stays, no result is made up.
		{Kind: "message", Role: "assistant", Model: "claude-test", Parts: []portable.Part{
			{Type: "tool_call", ID: "t2", Name: "Bash", Input: json.RawMessage(`{"command":"pwd"}`)},
		}},
	})
	if end.Dropped["thinking without text"] != 1 {
		t.Errorf("thinking without text = %d, want 1", end.Dropped["thinking without text"])
	}
	if end.Dropped["injected text"] != 3 {
		t.Errorf("injected text = %d, want 3", end.Dropped["injected text"])
	}
}

func TestExportCompactionRecord(t *testing.T) {
	_, _, records, _ := exportFile(t, "testdata/export/segments.jsonl", "seg")

	for _, r := range records {
		if r.Kind == "compaction" {
			if r.Role != "" || len(r.Parts) != 0 || r.Text != "Summary text" {
				t.Errorf("compaction record = %+v", r)
			}
			return
		}
	}
	t.Error("no compaction record")
}

// Only user turns carry injected blocks; the assistant writing about them
// must come through word for word.
func TestExportKeepsAssistantTags(t *testing.T) {
	_, _, records, end := exportFile(t, "testdata/export/assistant_tags.jsonl", "tags")

	want := "It adds <system-reminder>notes</system-reminder> and <command-name>/x</command-name>."
	if got := texts(records); len(got) != 2 || got[1] != want {
		t.Errorf("texts = %q, want the assistant's text unchanged", got)
	}
	if end.Dropped["injected text"] != 0 {
		t.Errorf("injected text = %d, want 0", end.Dropped["injected text"])
	}
}

// A prompt that talks about the tags is the user's own words, kept whole.
func TestExportKeepsTagMentionsInProse(t *testing.T) {
	_, _, records, end := exportFile(t, "testdata/export/prose.jsonl", "prose")
	var got []string
	for _, r := range records {
		for _, p := range r.Parts {
			got = append(got, p.Text)
		}
	}
	want := []string{
		"Please strip <system-reminder> blocks from user text, and keep everything after it.",
		"The <command-name> tag holds a slash command; keep this sentence.",
	}
	if !slices.Equal(got, want) || end.Dropped["injected text"] != 0 {
		t.Errorf("texts = %q, injected %d; want them unchanged", got, end.Dropped["injected text"])
	}
}
