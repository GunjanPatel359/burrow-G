package opencode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// fakeRun returns out for any command and remembers the last one.
type fakeRun struct {
	out  []byte
	err  error
	name string
	args []string
}

func (f *fakeRun) run(name string, args ...string) ([]byte, error) {
	f.name, f.args = name, args
	return f.out, f.err
}

func exportWith(t *testing.T, run func(string, ...string) ([]byte, error)) (portable.Header, []portable.Record, portable.End, error) {
	t.Helper()
	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	if err := (Source{Run: run}).Export(session.Session{Agent: "opencode", ID: "ses_test"}, w); err != nil {
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

func TestExportConverts(t *testing.T) {
	data, err := os.ReadFile("testdata/export.json")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeRun{out: data}
	h, records, end, err := exportWith(t, f.run)
	if err != nil {
		t.Fatal(err)
	}

	if f.name != "opencode" || !slices.Equal(f.args, []string{"session", "export", "--standalone", "ses_test"}) {
		t.Errorf("ran %s %q", f.name, f.args)
	}
	if h.Agent != "opencode" || h.ID != "ses_test" || h.Title != "Fix the bug" || h.Dir != "/tmp/p" {
		t.Errorf("header = %+v", h)
	}
	if !h.Created.Equal(time.UnixMilli(1793000000000)) || !h.Updated.Equal(time.UnixMilli(1793000060000)) {
		t.Errorf("header times = %v, %v", h.Created, h.Updated)
	}

	want := []portable.Record{
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "text", Text: "fix the bug [Image 1]"},
			{Type: "image", MediaType: "image/png", Data: "iVBO"},
		}},
		{Kind: "message", Role: "assistant", Model: "big-pickle", Parts: []portable.Part{
			{Type: "reasoning", Text: "think"},
			{Type: "text", Text: "on it"},
			{Type: "tool_call", ID: "t1", Name: "shell", Input: json.RawMessage(`{"command":"ls"}`)},
			{Type: "tool_call", ID: "t2", Name: "edit", Input: json.RawMessage(`{"path":"x.go"}`)},
			{Type: "tool_call", ID: "t3", Name: "read", Input: json.RawMessage(`{"path":"y.go"}`)},
		}},
		// t3 never finished: its call stays, no result is made up.
		{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "t1", Text: "a\nb\n"},
			{Type: "tool_result", ID: "t2", Text: "no match", Error: true},
		}},
		{Kind: "compaction", Text: "Sum"},
		{Kind: "message", Role: "assistant", Model: "big-pickle", Parts: []portable.Part{{Type: "text", Text: "done"}}},
	}
	if !reflect.DeepEqual(records, want) {
		got, _ := json.MarshalIndent(records, "", " ")
		t.Errorf("records:\n%s", got)
	}
	wantDropped := map[string]int{"injected text": 2, "bookkeeping line": 2, "attached file": 1}
	if !reflect.DeepEqual(end.Dropped, wantDropped) {
		t.Errorf("dropped = %v, want %v", end.Dropped, wantDropped)
	}
}

func TestExportNotOnPath(t *testing.T) {
	f := &fakeRun{err: fmt.Errorf("exec: %w", exec.ErrNotFound)}
	_, _, _, err := exportWith(t, f.run)
	if err == nil || err.Error() != "opencode not found on PATH" {
		t.Errorf("err = %v, want %q", err, "opencode not found on PATH")
	}
}

func TestExportCommandFails(t *testing.T) {
	f := &fakeRun{err: errors.New("exit status 1: Session not found")}
	_, _, _, err := exportWith(t, f.run)
	if err == nil || !strings.Contains(err.Error(), "opencode session export") || !strings.Contains(err.Error(), "Session not found") {
		t.Errorf("err = %v, want the command and its message", err)
	}
}

func TestExportBadOutput(t *testing.T) {
	f := &fakeRun{out: []byte("not json")}
	if _, _, _, err := exportWith(t, f.run); err == nil {
		t.Error("Export accepted output that isn't JSON")
	}
}

func TestExecReturnsWholeOutput(t *testing.T) {
	out, err := Exec("sh", "-c", "head -c 20000000 /dev/zero")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 20000000 {
		t.Errorf("got %d bytes, want 20000000", len(out))
	}
}

func TestExecErrorCarriesStderr(t *testing.T) {
	_, err := Exec("sh", "-c", "echo 'Session not found' >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "Session not found") {
		t.Errorf("err = %v, want stderr in it", err)
	}
}

func TestExecLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	if _, err := Exec("sh", "-c", "echo hi"); err != nil {
		t.Fatal(err)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("left behind %d files in TMPDIR", len(left))
	}
}
