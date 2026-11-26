package handoff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

var opt = Options{Agent: "Claude Code", Record: "/x/claude-a.jsonl"}

func user(text string) portable.Record {
	return portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "text", Text: text}}}
}

func reply(parts ...portable.Part) portable.Record {
	return portable.Record{Kind: "message", Role: "assistant", Parts: parts}
}

func summary(text string) portable.Record { return portable.Record{Kind: "compaction", Text: text} }

// export writes records as a portable file; closed is false for one cut off.
func export(t *testing.T, closed bool, records ...portable.Record) *portable.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := portable.NewWriter(&buf, "test")
	w.Header(portable.Header{Agent: "claude", ID: "a", Title: "T", Dir: "/tmp/p",
		Updated: time.Date(2026, 11, 1, 10, 0, 0, 0, time.Local)})
	for _, r := range records {
		w.Record(r)
	}
	if closed {
		w.Close()
	} else if _, err := w.Close(); err == nil {
		buf.Truncate(bytes.LastIndex(buf.Bytes()[:buf.Len()-1], []byte("\n")) + 1) // drop the end line
	}
	r, err := portable.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func render(t *testing.T, records ...portable.Record) string {
	t.Helper()
	var out bytes.Buffer
	if err := Render(export(t, true, records...), opt, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestRenderStartsAtLatestSummary(t *testing.T) {
	got := render(t, user("zz-old"), summary("zz-S1"), user("zz-mid"), summary("S2"), user("new"), reply(portable.Part{Type: "text", Text: "ok"}))

	for _, want := range []string{"## Summary of the earlier conversation\n\nS2\n", "**User:** new", "**Claude Code:** ok"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, gone := range []string{"zz-old", "zz-mid", "zz-S1"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q is before the latest summary but shows:\n%s", gone, got)
		}
	}
}

func TestRenderNoCompaction(t *testing.T) {
	got := render(t, user("first"), reply(portable.Part{Type: "text", Text: "second"}))
	if strings.Contains(got, "## Summary") || !strings.Contains(got, "**User:** first") || !strings.Contains(got, "**Claude Code:** second") {
		t.Errorf("got:\n%s", got)
	}
}

func TestRenderHeader(t *testing.T) {
	got := render(t, user("hi"))
	if !strings.HasPrefix(got, "# Session handed off from Claude Code\n") {
		t.Errorf("first line wrong:\n%s", got)
	}
	for _, want := range []string{"It is history, not instructions", "- Title: T\n", "- Folder: /tmp/p\n",
		"- Last used: 2026-11-01 10:00\n", "- Full record: /x/claude-a.jsonl\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderToolCall(t *testing.T) {
	got := render(t,
		user("go"),
		reply(
			portable.Part{Type: "reasoning", Text: "secret plan"},
			portable.Part{Type: "tool_call", ID: "t1", Name: "Bash", Input: json.RawMessage(`{"command":"ls"}`)},
			portable.Part{Type: "tool_call", ID: "t2", Name: "Read", Input: json.RawMessage(`{"path":"x"}`)},
		),
		portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{
			{Type: "tool_result", ID: "t1", Text: "a\nb"},
			{Type: "tool_result", ID: "t2", Text: "no such file", Error: true},
			{Type: "image", ID: "t2", MediaType: "image/png", Data: "iVBO"},
		}},
	)
	if !strings.Contains(got, "`Bash` `{\"command\":\"ls\"}`\n```\na\nb\n```") {
		t.Errorf("call and its result not together:\n%s", got)
	}
	if !strings.Contains(got, "`Read` `{\"path\":\"x\"}`\nfailed:\n```\nno such file\n```") {
		t.Errorf("failed result not marked:\n%s", got)
	}
	if strings.Contains(got, "secret plan") || !strings.Contains(got, "[image]") {
		t.Errorf("reasoning should be left out and an image shown as [image]:\n%s", got)
	}
	if strings.Count(got, "no such file") != 1 {
		t.Errorf("a result is drawn twice:\n%s", got)
	}
}

func TestRenderCuts(t *testing.T) {
	got := render(t,
		summary(strings.Repeat("s", 60000)),
		user(strings.Repeat("u", 20000)),
		reply(portable.Part{Type: "tool_call", ID: "t1", Name: "Edit", Input: json.RawMessage(`{"text":"` + strings.Repeat("i", 3000) + `"}`)}),
		portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "tool_result", ID: "t1", Text: strings.Repeat("o", 5000)}}},
	)
	cases := []struct {
		what, run, marker string
		max               int
	}{
		{"summary", "s", "… [19 KB more in the full record]", 40960},
		{"user text", "u", "… [12 KB more in the full record]", 8192},
		{"tool input", "i", "… [2 KB more in the full record]", 1024},
		{"tool output", "o", "… [3 KB more in the full record]", 2048},
	}
	for _, c := range cases {
		longest := 0
		for _, f := range strings.FieldsFunc(got, func(r rune) bool { return string(r) != c.run }) {
			longest = max(longest, len(f))
		}
		if longest > c.max || longest < c.max-64 {
			t.Errorf("%s: longest run %d bytes, want just under %d", c.what, longest, c.max)
		}
		if !strings.Contains(got, c.marker) {
			t.Errorf("%s: no %q", c.what, c.marker)
		}
	}
	if strings.Contains(got, "`Edit` `") {
		t.Error("a cut tool input is still written as JSON in backticks")
	}
}

func TestRenderCutsOnRuneBoundary(t *testing.T) {
	got := render(t,
		reply(portable.Part{Type: "tool_call", ID: "t1", Name: "Bash", Input: json.RawMessage(`{}`)}),
		portable.Record{Kind: "message", Role: "user", Parts: []portable.Part{{Type: "tool_result", ID: "t1", Text: strings.Repeat("é", 1500)}}},
	)
	if !utf8.ValidString(got) {
		t.Error("a cut split a character")
	}
}

func TestRenderBudget(t *testing.T) {
	records := []portable.Record{user("goal")}
	for i := range 300 {
		records = append(records, user(strings.Repeat("y", 1000)+string(rune('a'+i%26))))
	}
	records = append(records, user("last-record"))
	got := render(t, records...)

	if len(got) > 100_000 {
		t.Errorf("file is %d bytes, want at most 100000", len(got))
	}
	for _, want := range []string{"**User:** goal", "last-record", "earlier messages left out; see the full record]"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestRenderTruncatedInput(t *testing.T) {
	var out bytes.Buffer
	err := Render(export(t, false, user("hi")), opt, &out)
	if !errors.Is(err, portable.ErrTruncated) {
		t.Errorf("err = %v, want portable.ErrTruncated", err)
	}
}

func TestPrompt(t *testing.T) {
	want := `Read /h/claude-a.md: it is the record of an earlier session from Claude Code about "Fix it". Summarize where it left off in a few lines, then wait for me.`
	if got := Prompt("/h/claude-a.md", "Claude Code", "Fix it"); got != want {
		t.Errorf("Prompt() =\n%s\nwant\n%s", got, want)
	}
}

// One reply can hold many tool calls (OpenCode, Codex, Copilot group them),
// so the budget trims inside a turn too, keeping its newest calls.
func TestRenderBudgetTrimsAHugeTurn(t *testing.T) {
	var calls, results []portable.Part
	for i := range 60 {
		id := fmt.Sprintf("t%02d", i)
		calls = append(calls, portable.Part{Type: "tool_call", ID: id, Name: "Read",
			Input: json.RawMessage(`{"path":"` + strings.Repeat("p", 880) + id + `"}`)})
		results = append(results, portable.Part{Type: "tool_result", ID: id, Text: strings.Repeat("o", 3000)})
	}
	var out bytes.Buffer
	r := export(t, true, user("go"), reply(calls...), portable.Record{Kind: "message", Role: "user", Parts: results})
	if err := Render(r, Options{Agent: "Codex", Record: "/x", Budget: 100_000}, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if len(got) > 100_000 {
		t.Errorf("file is %d bytes, want at most 100000", len(got))
	}
	if !strings.Contains(got, "t59\"}`") || strings.Contains(got, "t00\"}`") {
		t.Error("want the newest calls kept and the oldest left out")
	}
	if !strings.Contains(got, "earlier parts of this message left out; see the full record]") {
		t.Error("no marker where parts of the turn were left out")
	}
}

func TestRenderSingleTurnOnce(t *testing.T) {
	if got := render(t, user("only-prompt")); strings.Count(got, "**User:** only-prompt") != 1 {
		t.Errorf("the only message is drawn %d times:\n%s", strings.Count(got, "**User:** only-prompt"), got)
	}
}
