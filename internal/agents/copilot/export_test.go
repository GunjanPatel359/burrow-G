package copilot

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// copilotHome builds ~/.copilot with a session-store.db holding ids, and
// copies events into session-state/<events id>/events.jsonl when set.
func copilotHome(t *testing.T, eventsID string) string {
	t.Helper()
	home := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(home, "session-store.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE sessions (id TEXT, cwd TEXT, summary TEXT, created_at TEXT, updated_at TEXT);
		INSERT INTO sessions VALUES ('cp1', '/tmp/x', 'Explain repo', '2026-11-04T10:00:00Z', '2026-11-04T10:00:13Z');
		INSERT INTO sessions VALUES ('empty', '/tmp/y', NULL, '2026-11-04T09:00:00Z', '2026-11-04T09:00:00Z');
	`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if eventsID != "" {
		data, err := os.ReadFile("testdata/export/events.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(home, "session-state", eventsID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func exportID(t *testing.T, home, id string) (portable.Header, []portable.Record, portable.End, error) {
	t.Helper()
	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	src := Source{DBPath: filepath.Join(home, "session-store.db")}
	if err := src.Export(session.Session{Agent: "copilot", ID: id}, w); err != nil {
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

func TestExportEvents(t *testing.T) {
	h, records, end, err := exportID(t, copilotHome(t, "cp1"), "cp1")
	if err != nil {
		t.Fatal(err)
	}
	if h.Agent != "copilot" || h.ID != "cp1" || h.Title != "Explain repo" || h.Dir != "/tmp/x" {
		t.Errorf("header = %+v", h)
	}
	if !h.Created.Equal(time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)) || !h.Updated.Equal(time.Date(2026, 11, 4, 10, 0, 13, 0, time.UTC)) {
		t.Errorf("header times = %v, %v", h.Created, h.Updated)
	}

	want := []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: "explain"}}},
		{Kind: "message", Role: "assistant", Model: "gpt-test", Parts: []portable.Part{
			{Type: "reasoning", Text: "plan"},
			{Type: "text", Text: "Looking."},
			{Type: "tool_call", ID: "t1", Name: "view", Input: json.RawMessage(`{"path":"/a"}`)},
			{Type: "tool_call", ID: "t2", Name: "task", Input: json.RawMessage(`{"prompt":"x"}`)},
		}},
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "t1", Text: "a.py"},
			{Type: "tool_result", ID: "t2", Text: "denied", Error: true},
		}},
		{Kind: "message", Role: "assistant", Model: "gpt-test", Parts: []portable.Part{{Type: "text", Text: "Done."}}},
	}
	if !reflect.DeepEqual(records, want) {
		got, _ := json.MarshalIndent(records, "", " ")
		t.Errorf("records:\n%s", got)
	}
	wantDropped := map[string]int{"injected text": 1, "attached file": 1, "sidechain line": 3, "thinking without text": 1, "bookkeeping line": 6}
	if !reflect.DeepEqual(end.Dropped, wantDropped) {
		t.Errorf("dropped = %v, want %v", end.Dropped, wantDropped)
	}
}

// A session Copilot never wrote events for is empty, not an error.
func TestExportNoEvents(t *testing.T) {
	h, records, _, err := exportID(t, copilotHome(t, ""), "empty")
	if err != nil {
		t.Fatal(err)
	}
	if h.Title != "(untitled)" || h.Dir != "/tmp/y" || len(records) != 0 {
		t.Errorf("header %+v, %d records", h, len(records))
	}
}

func TestExportBadID(t *testing.T) {
	home := copilotHome(t, "cp1")
	for _, id := range []string{"nope", "../cp1", "a/b", "", ".."} {
		if _, _, _, err := exportID(t, home, id); err == nil {
			t.Errorf("Export(%q) succeeded, want an error", id)
		}
	}
}
