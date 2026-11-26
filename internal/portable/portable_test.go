package portable

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, "9.9.9")
	if err := w.Header(Header{Agent: "claude", ID: "abc", Title: "T", Dir: "/tmp/p"}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	records := []Record{
		{Kind: "message", Role: "user", Time: at, Parts: []Part{{Type: "text", Text: "hi"}}},
		{Kind: "message", Role: "assistant", Time: at, Model: "m", Parts: []Part{
			{Type: "tool_call", ID: "t1", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`)},
		}},
	}
	for _, r := range records {
		if err := w.Record(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Drop("x")
	w.Drop("x")
	end, err := w.Close()
	if err != nil {
		t.Fatal(err)
	}
	wantEnd := End{Kind: "end", Records: 2, Dropped: map[string]int{"x": 2}}
	if !reflect.DeepEqual(end, wantEnd) {
		t.Errorf("Close() = %+v, want %+v", end, wantEnd)
	}

	r, err := NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	h := r.Header()
	if h.Kind != "header" || h.Format != "burrow-session/1" || h.Burrow != "9.9.9" || h.Exported.IsZero() {
		t.Errorf("header = %+v", h)
	}
	if h.Agent != "claude" || h.ID != "abc" || h.Title != "T" || h.Dir != "/tmp/p" {
		t.Errorf("header lost fields: %+v", h)
	}
	for i, want := range records {
		got, err := r.Next()
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("record %d = %+v\nwant %+v", i, got, want)
		}
	}
	if _, err := r.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("after last record: err = %v, want io.EOF", err)
	}
	if !reflect.DeepEqual(r.End(), wantEnd) {
		t.Errorf("End() = %+v, want %+v", r.End(), wantEnd)
	}
}

func TestHeaderKeepsExported(t *testing.T) {
	at := time.Date(2026, 11, 26, 6, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	w := NewWriter(&buf, "1")
	if err := w.Header(Header{Exported: at}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Header().Exported.Equal(at) {
		t.Errorf("Exported = %v, want %v", r.Header().Exported, at)
	}
}

func TestTruncated(t *testing.T) {
	in := `{"kind":"header","format":"burrow-session/1"}
{"kind":"message","role":"user","time":"2026-11-01T10:00:00Z","parts":[{"type":"text","text":"hi"}]}
`
	r, err := NewReader(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Next(); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if _, err := r.Next(); !errors.Is(err, ErrTruncated) {
		t.Errorf("err = %v, want ErrTruncated", err)
	}
}

func TestRefusesOtherFormat(t *testing.T) {
	for _, in := range []string{
		`{"kind":"header","format":"other/1"}`,
		`{"kind":"message","role":"user"}`,
		``,
	} {
		if _, err := NewReader(strings.NewReader(in + "\n")); err == nil {
			t.Errorf("NewReader(%q) accepted it", in)
		}
	}
}

func TestLongLine(t *testing.T) {
	long := strings.Repeat("a", 1<<20)
	var buf bytes.Buffer
	w := NewWriter(&buf, "1")
	w.Header(Header{})
	w.Record(Record{Kind: "message", Role: "user", Parts: []Part{{Type: "text", Text: long}}})
	if _, err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got.Parts[0].Text != long {
		t.Errorf("text is %d bytes, want %d", len(got.Parts[0].Text), len(long))
	}
}

func TestNoHTMLEscape(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, "1")
	w.Header(Header{})
	w.Record(Record{Kind: "message", Role: "user", Parts: []Part{{Type: "text", Text: "<a>&"}}})
	if _, err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "<a>&") {
		t.Errorf("text was escaped:\n%s", buf.String())
	}
}
