package qoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func exportFile(t *testing.T, fixture, id string) (string, portable.Header, []portable.Record, portable.End) {
	t.Helper()
	home := t.TempDir()
	copyFile(t, fixture, filepath.Join(home, "projects", "-tmp-p", id+".jsonl"))

	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	if err := (Source{Home: home}).Export(session.Session{Agent: "qoder", ID: id}, w); err != nil {
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
		rec.Time = time.Time{}
		records = append(records, rec)
	}
	return home, r.Header(), records, r.End()
}

func TestExportSegments(t *testing.T) {
	home, h, records, end := exportFile(t, "testdata/export/segments.jsonl", "seg")

	want := []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "first"}}},
		{Kind: "message", Role: "assistant", Model: "qoder-test", Parts: []portable.Part{
			{Type: "reasoning", Text: "plan"}, {Type: "text", Text: "one"},
		}},
		{Kind: "compaction", Text: "Summary text"},
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "second"}}},
		{Kind: "message", Role: "assistant", Model: "qoder-test", Parts: []portable.Part{{Type: "text", Text: "two"}}},
	}
	if !reflect.DeepEqual(records, want) {
		got, _ := json.MarshalIndent(records, "", " ")
		t.Errorf("records:\n%s", got)
	}
	// Qoder writes some timestamps as numbers: those lines are still read.
	if end.Dropped["unreadable line"] != 0 {
		t.Errorf("unreadable line = %d, want 0", end.Dropped["unreadable line"])
	}

	listed, err := Source{Home: home}.List()
	if err != nil || len(listed) != 1 {
		t.Fatalf("List() = %v, %v", listed, err)
	}
	if h.Agent != "qoder" || h.ID != "seg" || h.Title != listed[0].Title || h.Dir != listed[0].Dir {
		t.Errorf("header = %+v, List shows %+v", h, listed[0])
	}
	if want := time.Date(2026, 10, 26, 4, 53, 20, 0, time.UTC); !h.Created.Equal(want) {
		t.Errorf("Created = %v, want %v", h.Created, want)
	}
	if want := time.UnixMilli(1793000008000); !h.Updated.Equal(want) {
		t.Errorf("Updated = %v, want %v (the last line's numeric timestamp)", h.Updated, want)
	}
}

func TestExportGroupsParallelCalls(t *testing.T) {
	_, _, records, _ := exportFile(t, "testdata/export/parallel.jsonl", "par")

	var kinds []string
	for _, r := range records {
		var types []string
		for _, p := range r.Parts {
			types = append(types, p.Type+":"+p.ID)
		}
		kinds = append(kinds, r.Role+" "+string(mustJSON(types)))
	}
	want := []string{
		`user ["text:"]`,
		`assistant ["tool_call:ta","tool_call:tb"]`,
		`user ["tool_result:ta","tool_result:tb"]`,
		`assistant ["text:"]`,
	}
	if !slices.Equal(kinds, want) {
		t.Errorf("records = %q\nwant %q", kinds, want)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestExportRefusesPathID(t *testing.T) {
	home := t.TempDir()
	copyFile(t, "testdata/export/segments.jsonl", filepath.Join(home, "projects", "x.jsonl"))
	for _, id := range []string{"../x", "a/b", "*", "..", "", "nope"} {
		w := portable.NewWriter(io.Discard, "test")
		if err := (Source{Home: home}).Export(session.Session{ID: id}, w); err == nil {
			t.Errorf("Export(%q) succeeded, want an error", id)
		}
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
