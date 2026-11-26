package codex

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func exportID(t *testing.T, home, id string) (portable.Header, []portable.Record, portable.End, error) {
	t.Helper()
	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	if err := (Source{Home: home}).Export(session.Session{Agent: "codex", ID: id}, w); err != nil {
		return portable.Header{}, nil, portable.End{}, err
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
	return r.Header(), records, r.End(), nil
}

func TestExportRollout(t *testing.T) {
	h, records, end, err := exportID(t, "testdata/export", "cx")
	if err != nil {
		t.Fatal(err)
	}

	listed, err := Source{Home: "testdata/export"}.List()
	if err != nil || len(listed) != 1 {
		t.Fatalf("List() = %v, %v", listed, err)
	}
	if h.Agent != "codex" || h.ID != "cx" || h.Title != listed[0].Title || h.Dir != "/tmp/c" {
		t.Errorf("header = %+v, List shows %+v", h, listed[0])
	}
	if !h.Created.Equal(time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)) || !h.Updated.Equal(time.Date(2026, 11, 3, 10, 0, 15, 0, time.UTC)) {
		t.Errorf("header times = %v, %v", h.Created, h.Updated)
	}

	want := []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "text", Text: "add retry logic"},
			{Type: "image", MediaType: "image/png", Data: "iVBO"},
		}},
		{Kind: "message", Role: "assistant", Model: "gpt-test", Parts: []portable.Part{
			{Type: "reasoning", Text: "plan"},
			{Type: "text", Text: "on it"},
			{Type: "tool_call", ID: "c1", Name: "shell", Input: json.RawMessage(`{"command":["ls"]}`)},
			{Type: "tool_call", ID: "c2", Name: "apply_patch", Input: json.RawMessage(`"*** Begin Patch"`)},
		}},
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "c1", Text: "a\nb"},
			{Type: "tool_result", ID: "c2", Text: "patch failed", Error: true},
		}},
		{Kind: "message", Role: "assistant", Model: "gpt-test", Parts: []portable.Part{{Type: "text", Text: "done"}}},
		{Kind: "compaction", Text: "Sum"},
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "next"}}},
		// Cut off mid call: the call stays, no result is made up.
		{Kind: "message", Role: "assistant", Model: "gpt-test", Parts: []portable.Part{
			{Type: "tool_call", ID: "c3", Name: "shell", Input: json.RawMessage(`{}`)},
		}},
	}
	if !reflect.DeepEqual(records, want) {
		got, _ := json.MarshalIndent(records, "", " ")
		t.Errorf("records:\n%s", got)
	}
	if end.Dropped["injected text"] != 2 || end.Dropped["thinking without text"] != 1 || end.Dropped["bookkeeping line"] != 3 {
		t.Errorf("dropped = %v, want injected text 2, thinking without text 1, bookkeeping line 3", end.Dropped)
	}
}

func TestExportNotFound(t *testing.T) {
	if _, _, _, err := exportID(t, "testdata/export", "nope"); err == nil {
		t.Error("Export of an unknown id succeeded")
	}
}
