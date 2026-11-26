package claude

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
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
