package freebuff

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

const exportID = "2026-08-31T11-08-20.000Z"

// exportHome copies testdata/export into projects/db/chats/<exportID>/.
func exportHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "projects", "db", "chats", exportID)
	for _, name := range []string{"chat-messages.json", "run-state.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", "export", name))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, name), string(data))
	}
	writeFile(t, filepath.Join(dir, "chat-title.txt"), "Fix the login")
	if err := os.Chtimes(filepath.Join(dir, "chat-messages.json"), used, used); err != nil {
		t.Fatal(err)
	}
	return home
}

func export(t *testing.T, home string, s session.Session) (portable.Header, []portable.Record, portable.End, error) {
	t.Helper()
	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	if err := (Source{Home: home}).Export(s, w); err != nil {
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
		records = append(records, rec)
	}
	return r.Header(), records, r.End(), nil
}

func ms(n int64) time.Time { return time.UnixMilli(n).UTC() }

func TestExport(t *testing.T) {
	h, records, end, err := export(t, exportHome(t), session.Session{ID: exportID, Dir: "/w/db"})
	if err != nil {
		t.Fatal(err)
	}
	if h.Agent != "freebuff" || h.ID != exportID || h.Title != "Fix the login" || h.Dir != "/w/db" {
		t.Errorf("header = %+v", h)
	}
	if !h.Created.Equal(ms(1788174500000)) || !h.Updated.Equal(used) {
		t.Errorf("header times = %v, %v; want %v, %v", h.Created, h.Updated, ms(1788174500000), used)
	}

	// The writer compacts tool inputs; results are text and stay as written.
	raw := func(s string) json.RawMessage {
		var b bytes.Buffer
		if err := json.Compact(&b, []byte(s)); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	text := func(role string, at int64, s string) portable.Record {
		return portable.Record{Kind: "message", Role: role, Time: ms(at), Parts: []portable.Part{{Type: "text", Text: s}}}
	}
	turn := int64(1788174502000)
	want := []portable.Record{
		text("user", 1788174501000, "fix the login"),
		{Kind: "message", Role: "assistant", Time: ms(turn), Parts: []portable.Part{
			{Type: "reasoning", Text: "look first"},
			{Type: "text", Text: "Reading the code."},
			{Type: "tool_call", ID: "t1", Name: "read_files", Input: raw(`{"paths": ["a.go"]}`)},
			{Type: "tool_call", ID: "t2", Name: "code_search", Input: raw(`{"pattern": "login"}`)},
		}},
		{Kind: "message", Role: "user", Time: ms(turn), Parts: []portable.Part{
			{Type: "tool_result", ID: "t1", Text: "package a"},
			{Type: "tool_result", ID: "t2", Text: "a.go:3"},
		}},
		// Text after tool calls starts the next reply.
		{Kind: "message", Role: "assistant", Time: ms(turn), Parts: []portable.Part{
			{Type: "text", Text: "Found it. Asking you."},
			{Type: "tool_call", ID: "t3", Name: "ask_user", Input: raw(`{"questions":[{"question": "Which?", "options": [{"label": "A"}]}]}`)},
			{Type: "tool_call", ID: "t4-0", Name: "spawn_agents", Input: raw(`{"agents":[{"agent_type":"basher","prompt":"run tests","params":{"command": "go test"}}]}`)},
		}},
		{Kind: "message", Role: "user", Time: ms(turn), Parts: []portable.Part{
			{Type: "tool_result", ID: "t3", Text: `{"answers":[{"questionIndex":0,"selectedOption":"A"}]}`},
			{Type: "tool_result", ID: "t4-0", Text: "ok: 3 passed"},
		}},
		// t5 was still running: a call with no result.
		{Kind: "message", Role: "assistant", Time: ms(turn), Parts: []portable.Part{
			{Type: "text", Text: "Done."},
			{Type: "tool_call", ID: "t5", Name: "run_terminal_command", Input: raw(`{"command": "make"}`)},
		}},
		text("user", 1788174503000, "/compact"),
		text("assistant", 1788174504000, "Summarized."),
		{Kind: "compaction", Time: ms(1788174505000), Text: "Keep the login fix."},
		text("user", 1788174508000, "thanks"),
		text("assistant", 1788174509000, "You're welcome."),
	}
	if len(records) != len(want) {
		t.Errorf("got %d records, want %d", len(records), len(want))
	}
	for i := range min(len(records), len(want)) {
		if !reflect.DeepEqual(records[i], want[i]) {
			got, _ := json.Marshal(records[i])
			exp, _ := json.Marshal(want[i])
			t.Errorf("record %d:\n got %s\nwant %s", i, got, exp)
		}
	}
	wantDropped := map[string]int{"attached file": 2, "bookkeeping line": 4, "sidechain line": 2, "unreadable line": 1}
	if !reflect.DeepEqual(end.Dropped, wantDropped) {
		t.Errorf("dropped = %v, want %v", end.Dropped, wantDropped)
	}
}

// A chat never compacted has no summary. With no time in any message id,
// the header's Created falls back to last used and the records keep a zero
// time.
func TestExport_NoCompaction(t *testing.T) {
	home := t.TempDir()
	writeChat(t, home, chat{project: "db", id: "a", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: "hi"})
	h, records, _, err := export(t, home, session.Session{ID: "a", Dir: "/one/db"})
	if err != nil {
		t.Fatal(err)
	}
	want := []portable.Record{{Kind: "message", Role: "user", Time: time.Time{}, Parts: []portable.Part{{Type: "text", Text: "hi"}}}}
	if !reflect.DeepEqual(records, want) {
		t.Errorf("records = %+v", records)
	}
	if h.Title != "hi" || !h.Created.Equal(used) {
		t.Errorf("header = %+v", h)
	}
}

func TestExport_Empty(t *testing.T) {
	home := t.TempDir()
	writeChat(t, home, chat{project: "db", id: "a", runState: runState("/one/db", "/one/db"), messages: true})
	dir := filepath.Join(home, "projects", "db", "chats", "a")
	writeFile(t, filepath.Join(dir, "chat-messages.json"), "[]")
	h, records, _, err := export(t, home, session.Session{ID: "a", Dir: "/one/db"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 || h.Created.IsZero() || !h.Created.Equal(h.Updated) {
		t.Errorf("header %+v, %d records", h, len(records))
	}
}

func TestExport_NotFound(t *testing.T) {
	home := exportHome(t)
	for _, s := range []session.Session{
		{ID: "nope", Dir: "/w/db"},
		{ID: exportID, Dir: "/w/other"}, // the folder names the project
		{ID: "../" + exportID, Dir: "/w/db"},
		{ID: "a/b", Dir: "/w/db"},
		{ID: "..", Dir: "/w/db"},
		{ID: "", Dir: "/w/db"},
		{ID: exportID},
	} {
		if _, _, _, err := export(t, home, s); err == nil {
			t.Errorf("Export(%q in %q) succeeded, want an error", s.ID, s.Dir)
		}
	}
}

func TestExport_BrokenMessages(t *testing.T) {
	tests := []struct{ name, content string }{
		{"inside a message", `[{"id":"user-1788174501000","variant":"user","content":"hi"},{"id":`},
		{"before the closing bracket", `[{"id":"user-1788174501000","variant":"user","content":"hi"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := exportHome(t)
			path := filepath.Join(home, "projects", "db", "chats", exportID, "chat-messages.json")
			writeFile(t, path, tt.content)
			if _, _, _, err := export(t, home, session.Session{ID: exportID, Dir: "/w/db"}); err == nil {
				t.Error("want an error for a cut off chat-messages.json")
			}
		})
	}
}

// Freebuff tags the summary it writes; a prompt that only quotes the
// summary markup stays a message.
func TestExport_QuotedSummaryIsNotACompaction(t *testing.T) {
	home := exportHome(t)
	path := filepath.Join(home, "projects", "db", "chats", exportID, "run-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, strings.Replace(string(data), `"CONVERSATION_SUMMARY"`, `"USER_PROMPT"`, 1))
	_, records, _, err := export(t, home, session.Session{ID: exportID, Dir: "/w/db"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		if r.Kind == "compaction" {
			t.Errorf("an untagged message became a compaction: %+v", r)
		}
	}
}

func TestSummaryText(t *testing.T) {
	tests := []struct {
		name, content, want string
	}{
		{"parts", `[{"type":"text","text":"<conversation_summary> intro <historical_memory> the gist </historical_memory> </conversation_summary> more"}]`, "the gist"},
		{"string", `"<conversation_summary>just this</conversation_summary>"`, "just this"},
		{"not a summary", `[{"type":"text","text":"hello"}]`, ""},
		{"empty summary", `"<conversation_summary> <historical_memory> </historical_memory> </conversation_summary>"`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := summaryText(json.RawMessage(tt.content))
			if got != tt.want || ok != (tt.want != "") {
				t.Errorf("summaryText = %q, %v; want %q", got, ok, tt.want)
			}
		})
	}
}
